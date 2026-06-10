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
