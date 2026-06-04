package cache

import (
	"archive/zip"
	"os"
	"path/filepath"
	"testing"
)

func TestCacheStoreAndRetrieve(t *testing.T) {
	tmpDir := t.TempDir()
	c := New(tmpDir)
	c.Init()

	// Create a mock wheel file (zip)
	wheelPath := createMockWheel(t, tmpDir)

	// Store it
	key, err := c.Store(wheelPath)
	if err != nil {
		t.Fatalf("Store() error: %v", err)
	}

	if key.Algorithm != "sha256" {
		t.Errorf("algorithm = %q, want sha256", key.Algorithm)
	}
	if key.Digest == "" {
		t.Error("digest is empty")
	}

	// Verify it exists
	if !c.Has(key) {
		t.Error("Has() returned false after Store()")
	}

	// Retrieve it
	path, err := c.Retrieve(key)
	if err != nil {
		t.Fatalf("Retrieve() error: %v", err)
	}

	// Check extracted content exists
	extractedFile := filepath.Join(path, "test_pkg", "__init__.py")
	if _, err := os.Stat(extractedFile); err != nil {
		t.Errorf("extracted file not found: %v", err)
	}
}

func TestCacheIdempotentStore(t *testing.T) {
	tmpDir := t.TempDir()
	c := New(tmpDir)
	c.Init()

	wheelPath := createMockWheel(t, tmpDir)

	key1, err := c.Store(wheelPath)
	if err != nil {
		t.Fatal(err)
	}

	key2, err := c.Store(wheelPath)
	if err != nil {
		t.Fatal(err)
	}

	if key1.Digest != key2.Digest {
		t.Errorf("same wheel produced different keys: %s vs %s", key1.Digest, key2.Digest)
	}
}

func TestRefTracker(t *testing.T) {
	tmpDir := t.TempDir()
	c := New(tmpDir)
	c.Init()

	tracker := NewRefTracker(c)
	key := CASKey{Algorithm: "sha256", Digest: "abc123def456"}

	// Add reference
	err := tracker.AddReference("/path/to/env1", key, "numpy", "1.24.0")
	if err != nil {
		t.Fatalf("AddReference error: %v", err)
	}

	// Check count
	count, err := tracker.ReferenceCount(key)
	if err != nil {
		t.Fatal(err)
	}
	if count != 1 {
		t.Errorf("count = %d, want 1", count)
	}

	// Add another reference
	tracker.AddReference("/path/to/env2", key, "numpy", "1.24.0")
	count, _ = tracker.ReferenceCount(key)
	if count != 2 {
		t.Errorf("count = %d, want 2", count)
	}

	// Remove one
	tracker.RemoveReference("/path/to/env1", key)
	count, _ = tracker.ReferenceCount(key)
	if count != 1 {
		t.Errorf("count = %d, want 1", count)
	}

	// Remove all
	tracker.RemoveReference("/path/to/env2", key)
	count, _ = tracker.ReferenceCount(key)
	if count != 0 {
		t.Errorf("count = %d, want 0", count)
	}
}

func TestRefTrackerRemoveEnvironment(t *testing.T) {
	tmpDir := t.TempDir()
	c := New(tmpDir)
	c.Init()

	tracker := NewRefTracker(c)
	key1 := CASKey{Algorithm: "sha256", Digest: "aaa111"}
	key2 := CASKey{Algorithm: "sha256", Digest: "bbb222"}

	tracker.AddReference("/env/test", key1, "numpy", "1.24.0")
	tracker.AddReference("/env/test", key2, "scipy", "1.11.0")

	// Remove entire environment
	err := tracker.RemoveEnvironment("/env/test")
	if err != nil {
		t.Fatal(err)
	}

	count1, _ := tracker.ReferenceCount(key1)
	count2, _ := tracker.ReferenceCount(key2)
	if count1 != 0 || count2 != 0 {
		t.Errorf("counts after RemoveEnvironment: %d, %d (want 0, 0)", count1, count2)
	}
}

func createMockWheel(t *testing.T, dir string) string {
	t.Helper()
	wheelPath := filepath.Join(dir, "test_pkg-1.0.0-py3-none-any.whl")

	f, err := os.Create(wheelPath)
	if err != nil {
		t.Fatal(err)
	}

	w := zip.NewWriter(f)

	// Add a Python file
	fw, _ := w.Create("test_pkg/__init__.py")
	fw.Write([]byte("# test package\n"))

	// Add METADATA
	fw, _ = w.Create("test_pkg-1.0.0.dist-info/METADATA")
	fw.Write([]byte("Metadata-Version: 2.1\nName: test-pkg\nVersion: 1.0.0\n"))

	// Add RECORD
	fw, _ = w.Create("test_pkg-1.0.0.dist-info/RECORD")
	fw.Write([]byte("test_pkg/__init__.py,sha256=abc,15\n"))

	w.Close()
	f.Close()

	return wheelPath
}
