package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/kartikeyyadav/fpm/internal/snapshot"
	"github.com/kartikeyyadav/fpm/internal/venv"
	"github.com/spf13/cobra"
)

type bisectState struct {
	Active  bool     `json:"active"`
	Good    string   `json:"good"`
	Bad     string   `json:"bad"`
	Current string   `json:"current"`
	Tested  []string `json:"tested"`
	All     []string `json:"all"`
}

var bisectCmd = &cobra.Command{
	Use:   "bisect",
	Short: "Binary search through snapshots to find a breaking change",
	Long: `Find which snapshot introduced a problem by binary searching through history.

Usage:
  fpm bisect start
  fpm bisect good <snapshot-id>    Mark the current state as working
  fpm bisect bad <snapshot-id>     Mark the current state as broken
  fpm bisect reset                 Stop bisecting and return to original state`,
	GroupID: "advanced",
}

var bisectStartCmd = &cobra.Command{
	Use:   "start",
	Short: "Start a bisect session",
	RunE:  runBisectStart,
}

var bisectGoodCmd = &cobra.Command{
	Use:   "good [snapshot-id]",
	Short: "Mark a snapshot as good (working)",
	Args:  cobra.MaximumNArgs(1),
	RunE:  runBisectGood,
}

var bisectBadCmd = &cobra.Command{
	Use:   "bad [snapshot-id]",
	Short: "Mark a snapshot as bad (broken)",
	Args:  cobra.MaximumNArgs(1),
	RunE:  runBisectBad,
}

var bisectResetCmd = &cobra.Command{
	Use:   "reset",
	Short: "Stop bisecting",
	RunE:  runBisectReset,
}

func init() {
	bisectCmd.AddCommand(bisectStartCmd)
	bisectCmd.AddCommand(bisectGoodCmd)
	bisectCmd.AddCommand(bisectBadCmd)
	bisectCmd.AddCommand(bisectResetCmd)
	rootCmd.AddCommand(bisectCmd)
}

func bisectStatePath(envPath string) string {
	return filepath.Join(envPath, ".fpm-bisect.json")
}

func loadBisectState(envPath string) (*bisectState, error) {
	data, err := os.ReadFile(bisectStatePath(envPath))
	if err != nil {
		return &bisectState{}, nil
	}
	var state bisectState
	json.Unmarshal(data, &state)
	return &state, nil
}

func saveBisectState(envPath string, state *bisectState) error {
	data, _ := json.MarshalIndent(state, "", "  ")
	return os.WriteFile(bisectStatePath(envPath), data, 0644)
}

func runBisectStart(cmd *cobra.Command, args []string) error {
	cwd, _ := os.Getwd()
	activeVenv, _ := venv.Detect(cwd)
	if activeVenv == nil {
		return fmt.Errorf("no virtual environment found")
	}

	store := snapshot.NewStore(activeVenv.Path)
	snaps, err := store.List()
	if err != nil || len(snaps) < 2 {
		return fmt.Errorf("need at least 2 snapshots to bisect. Create more with 'fpm snapshot create'")
	}

	// Collect snapshot IDs (chronological order)
	var ids []string
	for _, s := range snaps {
		ids = append(ids, s.ID)
	}
	// Reverse to get oldest first
	for i, j := 0, len(ids)-1; i < j; i, j = i+1, j-1 {
		ids[i], ids[j] = ids[j], ids[i]
	}

	state := &bisectState{
		Active: true,
		All:    ids,
	}
	saveBisectState(activeVenv.Path, state)

	fmt.Printf("Bisect started with %d snapshots.\n", len(ids))
	fmt.Println("  Mark boundaries:")
	fmt.Println("    fpm bisect good <snapshot-id>   (last known working)")
	fmt.Println("    fpm bisect bad <snapshot-id>    (first known broken)")
	return nil
}

func runBisectGood(cmd *cobra.Command, args []string) error {
	cwd, _ := os.Getwd()
	activeVenv, _ := venv.Detect(cwd)
	if activeVenv == nil {
		return fmt.Errorf("no virtual environment found")
	}

	state, _ := loadBisectState(activeVenv.Path)
	if !state.Active {
		return fmt.Errorf("no bisect in progress. Run 'fpm bisect start' first")
	}

	if len(args) > 0 {
		state.Good = args[0]
	} else if state.Current != "" {
		state.Good = state.Current
		state.Tested = append(state.Tested, state.Current)
	} else {
		return fmt.Errorf("specify a snapshot-id or restore one first")
	}

	saveBisectState(activeVenv.Path, state)
	return bisectStep(activeVenv, state)
}

func runBisectBad(cmd *cobra.Command, args []string) error {
	cwd, _ := os.Getwd()
	activeVenv, _ := venv.Detect(cwd)
	if activeVenv == nil {
		return fmt.Errorf("no virtual environment found")
	}

	state, _ := loadBisectState(activeVenv.Path)
	if !state.Active {
		return fmt.Errorf("no bisect in progress. Run 'fpm bisect start' first")
	}

	if len(args) > 0 {
		state.Bad = args[0]
	} else if state.Current != "" {
		state.Bad = state.Current
		state.Tested = append(state.Tested, state.Current)
	} else {
		return fmt.Errorf("specify a snapshot-id or restore one first")
	}

	saveBisectState(activeVenv.Path, state)
	return bisectStep(activeVenv, state)
}

func bisectStep(activeVenv *venv.Venv, state *bisectState) error {
	if state.Good == "" || state.Bad == "" {
		fmt.Println("  Need both good and bad boundaries to begin bisecting.")
		return nil
	}

	// Find the range between good and bad
	goodIdx := -1
	badIdx := -1
	for i, id := range state.All {
		if id == state.Good {
			goodIdx = i
		}
		if id == state.Bad {
			badIdx = i
		}
	}

	if goodIdx == -1 || badIdx == -1 {
		return fmt.Errorf("good or bad snapshot not found in history")
	}
	if goodIdx > badIdx {
		goodIdx, badIdx = badIdx, goodIdx
	}

	// Find untested snapshots in range
	testedSet := make(map[string]bool)
	testedSet[state.Good] = true
	testedSet[state.Bad] = true
	for _, t := range state.Tested {
		testedSet[t] = true
	}

	var candidates []string
	for i := goodIdx + 1; i < badIdx; i++ {
		if !testedSet[state.All[i]] {
			candidates = append(candidates, state.All[i])
		}
	}

	if len(candidates) == 0 {
		// Found it!
		fmt.Printf("\n  \033[32mFound!\033[0m First bad snapshot: \033[1m%s\033[0m\n", state.Bad)
		fmt.Println("  Run 'fpm snapshot diff' to see what changed in that snapshot.")
		fmt.Println("  Run 'fpm bisect reset' to finish.")
		return nil
	}

	// Pick middle candidate
	mid := candidates[len(candidates)/2]
	state.Current = mid
	saveBisectState(activeVenv.Path, state)

	fmt.Printf("\n  Bisecting: %d snapshot(s) left to test\n", len(candidates))
	fmt.Printf("  Next: restore snapshot %s and test, then:\n", mid)
	fmt.Printf("    fpm snapshot restore %s\n", mid)
	fmt.Printf("    fpm bisect good     (if it works)\n")
	fmt.Printf("    fpm bisect bad      (if it's broken)\n")
	return nil
}

func runBisectReset(cmd *cobra.Command, args []string) error {
	cwd, _ := os.Getwd()
	activeVenv, _ := venv.Detect(cwd)
	if activeVenv == nil {
		return fmt.Errorf("no virtual environment found")
	}

	os.Remove(bisectStatePath(activeVenv.Path))
	fmt.Println("Bisect session ended.")
	return nil
}
