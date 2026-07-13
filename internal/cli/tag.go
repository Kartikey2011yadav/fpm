package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/kartikeyyadav/fpm/internal/venv"
	"github.com/spf13/cobra"
)

var tagCmd = &cobra.Command{
	Use:   "tag <name> [snapshot-id]",
	Short: "Create or manage named snapshot aliases",
	Long: `Create human-friendly names for snapshots.

Without a snapshot-id, tags the most recent snapshot.
Use 'fpm tag list' to see all tags, 'fpm tag delete <name>' to remove.

Tagged snapshots can be used with 'fpm snapshot restore <tag-name>'.`,
	GroupID: "environment",
	Args:    cobra.RangeArgs(1, 2),
	RunE:    runTag,
}

var tagListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all tags",
	RunE:  runTagList,
}

var tagDeleteCmd = &cobra.Command{
	Use:   "delete <name>",
	Short: "Delete a tag",
	Args:  cobra.ExactArgs(1),
	RunE:  runTagDelete,
}

func init() {
	tagCmd.AddCommand(tagListCmd)
	tagCmd.AddCommand(tagDeleteCmd)
	rootCmd.AddCommand(tagCmd)
}

func tagsFilePath(envPath string) string {
	snapshotDir := filepath.Join(envPath, ".fpm-snapshots")
	os.MkdirAll(snapshotDir, 0755)
	return filepath.Join(snapshotDir, "tags.json")
}

func loadTags(envPath string) (map[string]string, error) {
	path := tagsFilePath(envPath)
	data, err := os.ReadFile(path)
	if err != nil {
		return make(map[string]string), nil
	}
	tags := make(map[string]string)
	json.Unmarshal(data, &tags)
	return tags, nil
}

func saveTags(envPath string, tags map[string]string) error {
	path := tagsFilePath(envPath)
	data, err := json.MarshalIndent(tags, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}

func runTag(cmd *cobra.Command, args []string) error {
	cwd, _ := os.Getwd()
	activeVenv, _ := venv.Detect(cwd)
	if activeVenv == nil {
		return fmt.Errorf("no virtual environment found")
	}

	tagName := args[0]
	var snapshotID string

	if len(args) == 2 {
		snapshotID = args[1]
	} else {
		// Use most recent snapshot
		snapshotID = getMostRecentSnapshotID(activeVenv.Path)
		if snapshotID == "" {
			return fmt.Errorf("no snapshots found. Create one with 'fpm snapshot create'")
		}
	}

	tags, _ := loadTags(activeVenv.Path)
	tags[tagName] = snapshotID
	if err := saveTags(activeVenv.Path, tags); err != nil {
		return fmt.Errorf("failed to save tag: %w", err)
	}

	fmt.Printf("Tagged \"%s\" → %s\n", tagName, snapshotID)
	return nil
}

func runTagList(cmd *cobra.Command, args []string) error {
	cwd, _ := os.Getwd()
	activeVenv, _ := venv.Detect(cwd)
	if activeVenv == nil {
		return fmt.Errorf("no virtual environment found")
	}

	tags, _ := loadTags(activeVenv.Path)
	if len(tags) == 0 {
		fmt.Println("No tags.")
		return nil
	}

	fmt.Println("Tags:")
	for name, id := range tags {
		fmt.Printf("  \033[33m%s\033[0m → %s\n", name, id)
	}
	return nil
}

func runTagDelete(cmd *cobra.Command, args []string) error {
	cwd, _ := os.Getwd()
	activeVenv, _ := venv.Detect(cwd)
	if activeVenv == nil {
		return fmt.Errorf("no virtual environment found")
	}

	tagName := args[0]
	tags, _ := loadTags(activeVenv.Path)
	if _, exists := tags[tagName]; !exists {
		return fmt.Errorf("tag %q not found", tagName)
	}

	delete(tags, tagName)
	saveTags(activeVenv.Path, tags)
	fmt.Printf("Deleted tag \"%s\"\n", tagName)
	return nil
}

func getMostRecentSnapshotID(envPath string) string {
	snapshotDir := filepath.Join(envPath, ".fpm-snapshots")
	entries, err := os.ReadDir(snapshotDir)
	if err != nil {
		return ""
	}

	var latest string
	for _, e := range entries {
		name := e.Name()
		if len(name) > 5 && name[:5] == "snap-" {
			id := name[5:]
			if ext := filepath.Ext(id); ext != "" {
				id = id[:len(id)-len(ext)]
			}
			if id > latest {
				latest = id
			}
		}
	}
	return latest
}
