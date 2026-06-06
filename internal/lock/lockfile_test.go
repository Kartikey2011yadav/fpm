package lock

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/kartikeyyadav/fpm/internal/pep440"
	"github.com/kartikeyyadav/fpm/internal/pep508"
	"github.com/kartikeyyadav/fpm/internal/resolver"
	"github.com/kartikeyyadav/fpm/pkg/types"
)

func TestGenerate(t *testing.T) {
	res := &resolver.Resolution{
		Packages: []resolver.ResolvedPackage{
			{
				Name:    types.NewPackageName("requests"),
				Version: mustParse("2.31.0"),
				URL:     "https://files.pythonhosted.org/requests-2.31.0-py3-none-any.whl",
				Hash:    "abc123",
				Deps: []pep508.Requirement{
					{Name: types.NewPackageName("urllib3")},
				},
			},
			{
				Name:    types.NewPackageName("urllib3"),
				Version: mustParse("2.1.0"),
				URL:     "https://files.pythonhosted.org/urllib3-2.1.0-py3-none-any.whl",
			},
		},
	}

	lf := Generate(res, ">=3.10")

	if lf.Version != LockVersion {
		t.Errorf("version = %d, want %d", lf.Version, LockVersion)
	}
	if lf.RequiresPython != ">=3.10" {
		t.Errorf("requires-python = %q", lf.RequiresPython)
	}
	if len(lf.Packages) != 2 {
		t.Fatalf("packages count = %d, want 2", len(lf.Packages))
	}
}

func TestWriteAndRead(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "fpm.lock")

	lf := &Lockfile{
		Version:        1,
		RequiresPython: ">=3.10",
		Packages: []LockedPackage{
			{Name: "requests", Version: "2.31.0", Source: PackageSource{Registry: "https://pypi.org/simple"}},
			{Name: "urllib3", Version: "2.1.0", Source: PackageSource{Registry: "https://pypi.org/simple"}},
		},
	}

	if err := lf.Write(path); err != nil {
		t.Fatalf("Write error: %v", err)
	}

	read, err := Read(path)
	if err != nil {
		t.Fatalf("Read error: %v", err)
	}

	if len(read.Packages) != 2 {
		t.Errorf("read packages = %d, want 2", len(read.Packages))
	}
	if read.Packages[0].Name != "requests" {
		t.Errorf("first package = %q", read.Packages[0].Name)
	}
}

func TestDiff(t *testing.T) {
	old := &Lockfile{Packages: []LockedPackage{
		{Name: "requests", Version: "2.28.0"},
		{Name: "removed", Version: "1.0.0"},
	}}
	new := &Lockfile{Packages: []LockedPackage{
		{Name: "requests", Version: "2.31.0"},
		{Name: "added", Version: "1.0.0"},
	}}

	diff := Diff(old, new)
	if len(diff.Added) != 1 {
		t.Errorf("added = %d, want 1", len(diff.Added))
	}
	if len(diff.Removed) != 1 {
		t.Errorf("removed = %d, want 1", len(diff.Removed))
	}
	if len(diff.Changed) != 1 {
		t.Errorf("changed = %d, want 1", len(diff.Changed))
	}
}

func TestExportRequirementsTxt(t *testing.T) {
	lf := &Lockfile{Packages: []LockedPackage{
		{Name: "requests", Version: "2.31.0", Hashes: []string{"sha256:abc"}},
		{Name: "numpy", Version: "1.24.0"},
	}}

	output := lf.ExportRequirementsTxt()
	if !containsStr(output, "requests==2.31.0") {
		t.Error("missing requests==2.31.0")
	}
	if !containsStr(output, "numpy==1.24.0") {
		t.Error("missing numpy==1.24.0")
	}
	if !containsStr(output, "--hash=sha256:abc") {
		t.Error("missing hash")
	}
}

func TestFindPackage(t *testing.T) {
	lf := &Lockfile{Packages: []LockedPackage{
		{Name: "requests", Version: "2.31.0"},
	}}

	pkg := lf.FindPackage("requests")
	if pkg == nil {
		t.Fatal("FindPackage returned nil")
	}
	if pkg.Version != "2.31.0" {
		t.Errorf("version = %q", pkg.Version)
	}

	if lf.FindPackage("nonexistent") != nil {
		t.Error("expected nil for nonexistent package")
	}
}

func mustParse(s string) pep440.Version {
	v, _ := pep440.Parse(s)
	return v
}

func containsStr(s, sub string) bool {
	return len(s) >= len(sub) && (s == sub || len(s) > 0 && findStr(s, sub))
}

func findStr(s, sub string) bool {
	for i := 0; i <= len(s)-len(sub); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}

// Verify that unused imports are caught
var _ = os.Remove
