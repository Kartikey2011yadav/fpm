package snapshot

import (
	"testing"

	"github.com/kartikeyyadav/fpm/internal/env"
	"github.com/kartikeyyadav/fpm/internal/pep440"
	"github.com/kartikeyyadav/fpm/pkg/types"
)

func TestSnapshotCaptureAndRetrieve(t *testing.T) {
	tmpDir := t.TempDir()
	store := &Store{dir: tmpDir}

	scanResult := &env.ScanResult{
		Packages: []env.InstalledPackage{
			{Name: types.NewPackageName("requests"), Version: mustParse("2.31.0"), Manager: env.ManagerFpm},
			{Name: types.NewPackageName("numpy"), Version: mustParse("1.24.0"), Manager: env.ManagerPip},
		},
	}

	snap, err := store.Capture(scanResult, "3.11.8", "/usr/bin/python3", []string{"/site-packages"}, "test snapshot", "")
	if err != nil {
		t.Fatalf("Capture failed: %v", err)
	}

	if snap.ID == "" {
		t.Error("snapshot ID is empty")
	}
	if snap.Message != "test snapshot" {
		t.Errorf("message = %q, want 'test snapshot'", snap.Message)
	}
	if len(snap.Packages) != 2 {
		t.Errorf("packages count = %d, want 2", len(snap.Packages))
	}

	// Retrieve
	retrieved, err := store.Get(snap.ID)
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if retrieved.Message != "test snapshot" {
		t.Errorf("retrieved message = %q", retrieved.Message)
	}
}

func TestSnapshotList(t *testing.T) {
	tmpDir := t.TempDir()
	store := &Store{dir: tmpDir}

	scanResult := &env.ScanResult{Packages: []env.InstalledPackage{
		{Name: types.NewPackageName("pkg1"), Version: mustParse("1.0.0"), Manager: env.ManagerFpm},
	}}

	store.Capture(scanResult, "3.11", "/usr/bin/python3", nil, "first", "")
	store.Capture(scanResult, "3.11", "/usr/bin/python3", nil, "second", "")

	list, err := store.List()
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 2 {
		t.Errorf("list count = %d, want 2", len(list))
	}
}

func TestSnapshotDiff(t *testing.T) {
	old := &Snapshot{
		Packages: []SnapshotPackage{
			{Name: "requests", Version: "2.28.0", Manager: "fpm"},
			{Name: "numpy", Version: "1.24.0", Manager: "pip"},
			{Name: "removed-pkg", Version: "1.0.0", Manager: "fpm"},
		},
	}

	new := &Snapshot{
		Packages: []SnapshotPackage{
			{Name: "requests", Version: "2.31.0", Manager: "fpm"},
			{Name: "numpy", Version: "1.24.0", Manager: "pip"},
			{Name: "new-pkg", Version: "2.0.0", Manager: "fpm"},
		},
	}

	diff := DiffSnapshots(old, new)

	if len(diff.Added) != 1 || diff.Added[0].Name != "new-pkg" {
		t.Errorf("added = %v, want [new-pkg]", diff.Added)
	}
	if len(diff.Removed) != 1 || diff.Removed[0].Name != "removed-pkg" {
		t.Errorf("removed = %v, want [removed-pkg]", diff.Removed)
	}
	if len(diff.Changed) != 1 || diff.Changed[0].Name != "requests" {
		t.Errorf("changed = %v, want [requests]", diff.Changed)
	}
	if diff.Changed[0].OldVersion != "2.28.0" || diff.Changed[0].NewVersion != "2.31.0" {
		t.Errorf("version change = %s→%s, want 2.28.0→2.31.0",
			diff.Changed[0].OldVersion, diff.Changed[0].NewVersion)
	}
}

func TestSnapshotDelete(t *testing.T) {
	tmpDir := t.TempDir()
	store := &Store{dir: tmpDir}

	scanResult := &env.ScanResult{Packages: nil}
	snap, _ := store.Capture(scanResult, "3.11", "/usr/bin/python3", nil, "to delete", "")

	err := store.Delete(snap.ID)
	if err != nil {
		t.Fatalf("Delete failed: %v", err)
	}

	_, err = store.Get(snap.ID)
	if err == nil {
		t.Error("expected error after delete, got nil")
	}
}

func mustParse(s string) pep440.Version {
	v, err := pep440.Parse(s)
	if err != nil {
		panic(err)
	}
	return v
}

func TestSnapshot_EmptyEnvironment(t *testing.T) {
	tmpDir := t.TempDir()
	store := &Store{dir: tmpDir}

	scanResult := &env.ScanResult{Packages: nil}
	snap, err := store.Capture(scanResult, "3.12.0", "/usr/bin/python3", nil, "empty env", "")
	if err != nil {
		t.Fatalf("Capture empty env failed: %v", err)
	}
	if len(snap.Packages) != 0 {
		t.Errorf("expected 0 packages in empty snapshot, got %d", len(snap.Packages))
	}
	if snap.PythonVersion != "3.12.0" {
		t.Errorf("python version = %q, want 3.12.0", snap.PythonVersion)
	}
}

func TestSnapshot_DiffVersionChange(t *testing.T) {
	old := &Snapshot{
		Packages: []SnapshotPackage{
			{Name: "flask", Version: "2.3.0", Manager: "fpm"},
			{Name: "werkzeug", Version: "2.3.0", Manager: "fpm"},
		},
	}
	new := &Snapshot{
		Packages: []SnapshotPackage{
			{Name: "flask", Version: "3.0.0", Manager: "fpm"},
			{Name: "werkzeug", Version: "3.0.0", Manager: "fpm"},
		},
	}

	diff := DiffSnapshots(old, new)

	if len(diff.Changed) != 2 {
		t.Fatalf("expected 2 changed, got %d", len(diff.Changed))
	}
	if len(diff.Added) != 0 {
		t.Errorf("expected 0 added, got %d", len(diff.Added))
	}
	if len(diff.Removed) != 0 {
		t.Errorf("expected 0 removed, got %d", len(diff.Removed))
	}
}

func TestSnapshot_MultipleRestorePoints(t *testing.T) {
	tmpDir := t.TempDir()
	store := &Store{dir: tmpDir}

	// Create 3 snapshots with increasing packages
	scan1 := &env.ScanResult{Packages: []env.InstalledPackage{
		{Name: types.NewPackageName("flask"), Version: mustParse("3.0.0"), Manager: env.ManagerFpm},
	}}
	snap1, _ := store.Capture(scan1, "3.12", "/usr/bin/python3", nil, "v1", "")

	scan2 := &env.ScanResult{Packages: []env.InstalledPackage{
		{Name: types.NewPackageName("flask"), Version: mustParse("3.0.0"), Manager: env.ManagerFpm},
		{Name: types.NewPackageName("requests"), Version: mustParse("2.31.0"), Manager: env.ManagerFpm},
	}}
	snap2, _ := store.Capture(scan2, "3.12", "/usr/bin/python3", nil, "v2", "")

	scan3 := &env.ScanResult{Packages: []env.InstalledPackage{
		{Name: types.NewPackageName("flask"), Version: mustParse("3.0.0"), Manager: env.ManagerFpm},
		{Name: types.NewPackageName("requests"), Version: mustParse("2.31.0"), Manager: env.ManagerFpm},
		{Name: types.NewPackageName("numpy"), Version: mustParse("1.24.0"), Manager: env.ManagerFpm},
	}}
	store.Capture(scan3, "3.12", "/usr/bin/python3", nil, "v3", "")

	// Can retrieve any snapshot
	retrieved1, err := store.Get(snap1.ID)
	if err != nil {
		t.Fatalf("Get snap1 failed: %v", err)
	}
	if len(retrieved1.Packages) != 1 {
		t.Errorf("snap1 should have 1 package, got %d", len(retrieved1.Packages))
	}

	retrieved2, err := store.Get(snap2.ID)
	if err != nil {
		t.Fatalf("Get snap2 failed: %v", err)
	}
	if len(retrieved2.Packages) != 2 {
		t.Errorf("snap2 should have 2 packages, got %d", len(retrieved2.Packages))
	}

	// List shows all 3
	list, _ := store.List()
	if len(list) != 3 {
		t.Errorf("expected 3 snapshots, got %d", len(list))
	}
}

func TestSnapshot_DiffNoChanges(t *testing.T) {
	snap := &Snapshot{
		Packages: []SnapshotPackage{
			{Name: "flask", Version: "3.0.0", Manager: "fpm"},
			{Name: "requests", Version: "2.31.0", Manager: "fpm"},
		},
	}

	diff := DiffSnapshots(snap, snap)

	if len(diff.Added) != 0 || len(diff.Removed) != 0 || len(diff.Changed) != 0 {
		t.Errorf("diff of identical snapshots should be empty, got added=%d removed=%d changed=%d",
			len(diff.Added), len(diff.Removed), len(diff.Changed))
	}
}

func TestSnapshot_DiffAdditionsOnly(t *testing.T) {
	old := &Snapshot{Packages: []SnapshotPackage{
		{Name: "flask", Version: "3.0.0", Manager: "fpm"},
	}}
	new := &Snapshot{Packages: []SnapshotPackage{
		{Name: "flask", Version: "3.0.0", Manager: "fpm"},
		{Name: "requests", Version: "2.31.0", Manager: "fpm"},
		{Name: "numpy", Version: "1.24.0", Manager: "pip"},
	}}

	diff := DiffSnapshots(old, new)
	if len(diff.Added) != 2 {
		t.Errorf("expected 2 additions, got %d", len(diff.Added))
	}
	if len(diff.Removed) != 0 {
		t.Errorf("expected 0 removals, got %d", len(diff.Removed))
	}
}

func TestSnapshot_DiffRemovalsOnly(t *testing.T) {
	old := &Snapshot{Packages: []SnapshotPackage{
		{Name: "flask", Version: "3.0.0", Manager: "fpm"},
		{Name: "requests", Version: "2.31.0", Manager: "fpm"},
		{Name: "numpy", Version: "1.24.0", Manager: "pip"},
	}}
	new := &Snapshot{Packages: []SnapshotPackage{
		{Name: "flask", Version: "3.0.0", Manager: "fpm"},
	}}

	diff := DiffSnapshots(old, new)
	if len(diff.Removed) != 2 {
		t.Errorf("expected 2 removals, got %d", len(diff.Removed))
	}
	if len(diff.Added) != 0 {
		t.Errorf("expected 0 additions, got %d", len(diff.Added))
	}
}
