package snapshot

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/kartikeyyadav/fpm/internal/cache"
	"github.com/kartikeyyadav/fpm/internal/config"
	"github.com/kartikeyyadav/fpm/internal/env"
	"github.com/kartikeyyadav/fpm/internal/pep440"
	"github.com/kartikeyyadav/fpm/pkg/types"
)

type Snapshot struct {
	ID            string            `json:"id"`
	CreatedAt     time.Time         `json:"created_at"`
	Message       string            `json:"message,omitempty"`
	PythonVersion string            `json:"python_version"`
	PythonPath    string            `json:"python_path"`
	PathOrder     []string          `json:"path_order"`
	Packages      []SnapshotPackage `json:"packages"`
	FpmToml       string            `json:"fpm_toml,omitempty"`
}

type SnapshotPackage struct {
	Name     string `json:"name"`
	Version  string `json:"version"`
	Manager  string `json:"manager"`
	CASKey   string `json:"cas_key,omitempty"`
	Location string `json:"location"`
}

type SnapshotDiff struct {
	Added   []SnapshotPackage
	Removed []SnapshotPackage
	Changed []PackageVersionChange
}

type PackageVersionChange struct {
	Name       string
	OldVersion string
	NewVersion string
	Manager    string
}

type Store struct {
	dir string
}

func NewStore(envPath string) *Store {
	hash := hashPath(envPath)
	storeDir := filepath.Join(cacheBaseDir(), "snapshots", hash[:16])
	return &Store{dir: storeDir}
}

func (s *Store) Capture(scanResult *env.ScanResult, pythonVersion, pythonPath string, pathOrder []string, message string, projectDir string) (*Snapshot, error) {
	if err := os.MkdirAll(s.dir, 0755); err != nil {
		return nil, err
	}

	snap := &Snapshot{
		ID:            generateID(),
		CreatedAt:     time.Now().UTC(),
		Message:       message,
		PythonVersion: pythonVersion,
		PythonPath:    pythonPath,
		PathOrder:     pathOrder,
	}

	// Capture fpm.toml if it exists in the project
	if projectDir != "" {
		if data, err := os.ReadFile(filepath.Join(projectDir, "fpm.toml")); err == nil {
			snap.FpmToml = string(data)
		}
	}

	// Capture ALL packages from all managers
	for _, pkg := range scanResult.Packages {
		sp := SnapshotPackage{
			Name:     pkg.Name.Normalized(),
			Version:  pkg.Version.String(),
			Manager:  pkg.Manager.String(),
			Location: pkg.Location,
		}
		snap.Packages = append(snap.Packages, sp)
	}

	// Sort for deterministic output
	sort.Slice(snap.Packages, func(i, j int) bool {
		return snap.Packages[i].Name < snap.Packages[j].Name
	})

	// Write snapshot file
	if err := s.writeSnapshot(snap); err != nil {
		return nil, err
	}

	// Update current pointer
	if err := s.SetCurrent(snap.ID); err != nil {
		return nil, err
	}

	return snap, nil
}

func (s *Store) CaptureWithCAS(scanResult *env.ScanResult, refTracker *cache.RefTracker, pythonVersion, pythonPath string, pathOrder []string, message string, projectDir string) (*Snapshot, error) {
	if err := os.MkdirAll(s.dir, 0755); err != nil {
		return nil, err
	}

	snap := &Snapshot{
		ID:            generateID(),
		CreatedAt:     time.Now().UTC(),
		Message:       message,
		PythonVersion: pythonVersion,
		PythonPath:    pythonPath,
		PathOrder:     pathOrder,
	}

	// Capture fpm.toml if it exists in the project
	if projectDir != "" {
		if data, err := os.ReadFile(filepath.Join(projectDir, "fpm.toml")); err == nil {
			snap.FpmToml = string(data)
		}
	}

	for _, pkg := range scanResult.Packages {
		sp := SnapshotPackage{
			Name:     pkg.Name.Normalized(),
			Version:  pkg.Version.String(),
			Manager:  pkg.Manager.String(),
			Location: pkg.Location,
		}
		if pkg.Manager == env.ManagerFpm {
			sp.CASKey = lookupCASKey(refTracker, pkg.Name, pkg.Version)
		}
		snap.Packages = append(snap.Packages, sp)
	}

	sort.Slice(snap.Packages, func(i, j int) bool {
		return snap.Packages[i].Name < snap.Packages[j].Name
	})

	if err := s.writeSnapshot(snap); err != nil {
		return nil, err
	}
	if err := s.SetCurrent(snap.ID); err != nil {
		return nil, err
	}

	return snap, nil
}

func (s *Store) List() ([]*Snapshot, error) {
	entries, err := os.ReadDir(s.dir)
	if err != nil {
		return nil, nil
	}

	var snapshots []*Snapshot
	for _, entry := range entries {
		if !strings.HasPrefix(entry.Name(), "snap-") || !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}
		snap, err := s.readSnapshot(strings.TrimSuffix(entry.Name(), ".json"))
		if err != nil {
			continue
		}
		snapshots = append(snapshots, snap)
	}

	// Sort by creation time descending (newest first)
	sort.Slice(snapshots, func(i, j int) bool {
		return snapshots[i].CreatedAt.After(snapshots[j].CreatedAt)
	})

	return snapshots, nil
}

func (s *Store) Get(id string) (*Snapshot, error) {
	return s.readSnapshot("snap-" + id)
}

func (s *Store) Current() (*Snapshot, error) {
	currentID, err := s.GetCurrent()
	if err != nil {
		return nil, err
	}
	return s.Get(currentID)
}

func (s *Store) Delete(id string) error {
	path := filepath.Join(s.dir, "snap-"+id+".json")
	return os.Remove(path)
}

func (s *Store) Diff(id1, id2 string) (*SnapshotDiff, error) {
	snap1, err := s.Get(id1)
	if err != nil {
		return nil, fmt.Errorf("snapshot %s not found: %w", id1, err)
	}
	snap2, err := s.Get(id2)
	if err != nil {
		return nil, fmt.Errorf("snapshot %s not found: %w", id2, err)
	}

	return DiffSnapshots(snap1, snap2), nil
}

func DiffSnapshots(old, new *Snapshot) *SnapshotDiff {
	diff := &SnapshotDiff{}

	oldMap := make(map[string]SnapshotPackage)
	for _, p := range old.Packages {
		oldMap[p.Name] = p
	}

	newMap := make(map[string]SnapshotPackage)
	for _, p := range new.Packages {
		newMap[p.Name] = p
	}

	for name, newPkg := range newMap {
		if oldPkg, exists := oldMap[name]; exists {
			if oldPkg.Version != newPkg.Version {
				diff.Changed = append(diff.Changed, PackageVersionChange{
					Name:       name,
					OldVersion: oldPkg.Version,
					NewVersion: newPkg.Version,
					Manager:    newPkg.Manager,
				})
			}
		} else {
			diff.Added = append(diff.Added, newPkg)
		}
	}

	for name, oldPkg := range oldMap {
		if _, exists := newMap[name]; !exists {
			diff.Removed = append(diff.Removed, oldPkg)
		}
	}

	return diff
}

// DriftReport compares a snapshot against the current environment state
type DriftReport struct {
	Matching []SnapshotPackage
	Drifted  []DriftEntry
	Missing  []SnapshotPackage
	New      []env.InstalledPackage
}

type DriftEntry struct {
	Package        SnapshotPackage
	CurrentVersion string
	CurrentManager string
}

func (s *Store) DetectDrift(snapID string, currentScan *env.ScanResult) (*DriftReport, error) {
	snap, err := s.Get(snapID)
	if err != nil {
		return nil, err
	}

	report := &DriftReport{}

	// Build current state map
	currentMap := make(map[string]env.InstalledPackage)
	for _, pkg := range currentScan.Packages {
		currentMap[pkg.Name.Normalized()] = pkg
	}

	// Check each snapshot package against current state
	snapshotNames := make(map[string]bool)
	for _, sp := range snap.Packages {
		snapshotNames[sp.Name] = true
		current, exists := currentMap[sp.Name]

		if !exists {
			report.Missing = append(report.Missing, sp)
			continue
		}

		if current.Version.String() == sp.Version {
			report.Matching = append(report.Matching, sp)
		} else {
			report.Drifted = append(report.Drifted, DriftEntry{
				Package:        sp,
				CurrentVersion: current.Version.String(),
				CurrentManager: current.Manager.String(),
			})
		}
	}

	// Find packages in current env that weren't in snapshot
	for _, pkg := range currentScan.Packages {
		if !snapshotNames[pkg.Name.Normalized()] {
			report.New = append(report.New, pkg)
		}
	}

	return report, nil
}

// Internal helpers

func (s *Store) writeSnapshot(snap *Snapshot) error {
	data, err := json.MarshalIndent(snap, "", "  ")
	if err != nil {
		return err
	}
	path := filepath.Join(s.dir, "snap-"+snap.ID+".json")
	return os.WriteFile(path, data, 0644)
}

func (s *Store) readSnapshot(filename string) (*Snapshot, error) {
	path := filepath.Join(s.dir, filename+".json")
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var snap Snapshot
	return &snap, json.Unmarshal(data, &snap)
}

func (s *Store) SetCurrent(id string) error {
	path := filepath.Join(s.dir, "current")
	return os.WriteFile(path, []byte(id), 0644)
}

func (s *Store) GetCurrent() (string, error) {
	path := filepath.Join(s.dir, "current")
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(data)), nil
}

var idCounter int64

func generateID() string {
	now := time.Now()
	idCounter++
	return fmt.Sprintf("%s-%03d", now.Format("20060102-150405"), idCounter)
}

func hashPath(path string) string {
	h := sha256.Sum256([]byte(path))
	return hex.EncodeToString(h[:])
}

func cacheBaseDir() string {
	return config.CacheDir()
}

func lookupCASKey(tracker *cache.RefTracker, name types.PackageName, version pep440.Version) string {
	return tracker.FindCASKeyByPackage(name.Normalized(), version.String())
}
