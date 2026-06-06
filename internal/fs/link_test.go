package fs

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLinkFileCopy(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "src.txt")
	dst := filepath.Join(dir, "dst.txt")

	os.WriteFile(src, []byte("hello world"), 0644)

	err := LinkFile(src, dst, LinkModeCopy)
	if err != nil {
		t.Fatalf("LinkFile(copy) error: %v", err)
	}

	data, err := os.ReadFile(dst)
	if err != nil {
		t.Fatalf("ReadFile error: %v", err)
	}
	if string(data) != "hello world" {
		t.Errorf("content = %q", data)
	}
}

func TestLinkFileHardlink(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "src.txt")
	dst := filepath.Join(dir, "dst.txt")

	os.WriteFile(src, []byte("linked content"), 0644)

	err := LinkFile(src, dst, LinkModeHardlink)
	if err != nil {
		t.Fatalf("LinkFile(hardlink) error: %v", err)
	}

	data, _ := os.ReadFile(dst)
	if string(data) != "linked content" {
		t.Errorf("content = %q", data)
	}

	// Verify same inode
	srcInfo, _ := os.Stat(src)
	dstInfo, _ := os.Stat(dst)
	if !os.SameFile(srcInfo, dstInfo) {
		t.Error("files are not hardlinked (different inodes)")
	}
}

func TestLinkDir(t *testing.T) {
	dir := t.TempDir()
	srcDir := filepath.Join(dir, "src")
	dstDir := filepath.Join(dir, "dst")

	os.MkdirAll(filepath.Join(srcDir, "sub"), 0755)
	os.WriteFile(filepath.Join(srcDir, "a.txt"), []byte("a"), 0644)
	os.WriteFile(filepath.Join(srcDir, "sub", "b.txt"), []byte("b"), 0644)

	err := LinkDir(srcDir, dstDir, LinkModeCopy)
	if err != nil {
		t.Fatalf("LinkDir error: %v", err)
	}

	// Verify files exist in destination
	data, _ := os.ReadFile(filepath.Join(dstDir, "a.txt"))
	if string(data) != "a" {
		t.Errorf("a.txt = %q", data)
	}
	data2, _ := os.ReadFile(filepath.Join(dstDir, "sub", "b.txt"))
	if string(data2) != "b" {
		t.Errorf("sub/b.txt = %q", data2)
	}
}

func TestDirSize(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "file1"), []byte("12345"), 0644)
	os.WriteFile(filepath.Join(dir, "file2"), []byte("67890"), 0644)

	size, err := DirSize(dir)
	if err != nil {
		t.Fatalf("DirSize error: %v", err)
	}
	if size != 10 {
		t.Errorf("size = %d, want 10", size)
	}
}

func TestBestLinkMode(t *testing.T) {
	mode := BestLinkMode()
	if mode != LinkModeReflink && mode != LinkModeHardlink {
		t.Errorf("unexpected mode: %v", mode)
	}
}
