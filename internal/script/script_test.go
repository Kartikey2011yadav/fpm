package script

import (
	"os"
	"path/filepath"
	"testing"
)

func TestParseInlineMetadata(t *testing.T) {
	dir := t.TempDir()
	scriptPath := filepath.Join(dir, "test.py")
	os.WriteFile(scriptPath, []byte(`#!/usr/bin/env python3
# /// script
# dependencies = ["requests>=2.28", "rich"]
# requires-python = ">=3.10"
# ///

import requests
print("hello")
`), 0644)

	meta, err := ParseInlineMetadata(scriptPath)
	if err != nil {
		t.Fatalf("ParseInlineMetadata error: %v", err)
	}

	if len(meta.Dependencies) != 2 {
		t.Fatalf("deps count = %d, want 2", len(meta.Dependencies))
	}
	if meta.Dependencies[0] != "requests>=2.28" {
		t.Errorf("deps[0] = %q", meta.Dependencies[0])
	}
	if meta.Dependencies[1] != "rich" {
		t.Errorf("deps[1] = %q", meta.Dependencies[1])
	}
	if meta.RequiresPython != ">=3.10" {
		t.Errorf("requires-python = %q", meta.RequiresPython)
	}
}

func TestParseNoMetadata(t *testing.T) {
	dir := t.TempDir()
	scriptPath := filepath.Join(dir, "plain.py")
	os.WriteFile(scriptPath, []byte(`print("no metadata")`), 0644)

	_, err := ParseInlineMetadata(scriptPath)
	if err == nil {
		t.Error("expected error for script without metadata")
	}
}

func TestHasInlineMetadata(t *testing.T) {
	dir := t.TempDir()

	withMeta := filepath.Join(dir, "with.py")
	os.WriteFile(withMeta, []byte("# /// script\n# dependencies = [\"x\"]\n# ///\n"), 0644)

	without := filepath.Join(dir, "without.py")
	os.WriteFile(without, []byte("print('hi')\n"), 0644)

	if !HasInlineMetadata(withMeta) {
		t.Error("expected true for script with metadata")
	}
	if HasInlineMetadata(without) {
		t.Error("expected false for script without metadata")
	}
}
