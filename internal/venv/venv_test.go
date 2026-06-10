package venv

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func testBinDir() string {
	if runtime.GOOS == "windows" {
		return "Scripts"
	}
	return "bin"
}

func testPythonBin() string {
	if runtime.GOOS == "windows" {
		return "python.exe"
	}
	return "python3"
}

func TestDetect_FindsVenvInCwd(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a fake venv structure
	venvPath := filepath.Join(tmpDir, ".venv")
	binDir := filepath.Join(venvPath, testBinDir())
	os.MkdirAll(binDir, 0755)
	os.WriteFile(filepath.Join(venvPath, "pyvenv.cfg"), []byte("home = /usr/bin\n"), 0644)

	// Create a fake python3 binary
	os.WriteFile(filepath.Join(binDir, testPythonBin()), []byte("#!/bin/sh\n"), 0755)

	v, err := Detect(tmpDir)
	if err != nil {
		t.Fatalf("Detect should find .venv in cwd: %v", err)
	}
	if v.Path != venvPath {
		t.Errorf("expected path %q, got %q", venvPath, v.Path)
	}
}

func TestDetect_FindsVenvByWalkingUp(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a .venv at the project root
	venvPath := filepath.Join(tmpDir, ".venv")
	binDir := filepath.Join(venvPath, testBinDir())
	os.MkdirAll(binDir, 0755)
	os.WriteFile(filepath.Join(venvPath, "pyvenv.cfg"), []byte("home = /usr/bin\n"), 0644)
	os.WriteFile(filepath.Join(binDir, testPythonBin()), []byte("#!/bin/sh\n"), 0755)

	// Create subdirectory within the project
	subDir := filepath.Join(tmpDir, "src", "pkg")
	os.MkdirAll(subDir, 0755)

	// Detect from subdirectory should walk up and find .venv
	v, err := Detect(subDir)
	if err != nil {
		t.Fatalf("Detect should walk up and find venv: %v", err)
	}
	if v.Path != venvPath {
		t.Errorf("expected path %q, got %q", venvPath, v.Path)
	}
}

func TestDetect_NoVenvReturnsError(t *testing.T) {
	tmpDir := t.TempDir()

	_, err := Detect(tmpDir)
	if err == nil {
		t.Error("Detect should return error when no venv found")
	}
}

func TestDetect_DoesNotDetectSiblingVenv(t *testing.T) {
	tmpDir := t.TempDir()

	// Create two project directories
	projectA := filepath.Join(tmpDir, "project-a")
	projectB := filepath.Join(tmpDir, "project-b")
	os.MkdirAll(projectA, 0755)
	os.MkdirAll(projectB, 0755)

	// Only project-a has a venv
	venvPath := filepath.Join(projectA, ".venv")
	binDir := filepath.Join(venvPath, testBinDir())
	os.MkdirAll(binDir, 0755)
	os.WriteFile(filepath.Join(venvPath, "pyvenv.cfg"), []byte("home = /usr/bin\n"), 0644)
	os.WriteFile(filepath.Join(binDir, testPythonBin()), []byte("#!/bin/sh\n"), 0755)

	// Detect from project-b should NOT find project-a's venv
	_, err := Detect(projectB)
	if err == nil {
		t.Error("Detect in project-b should NOT find project-a's venv")
	}
}

func TestDetect_IsolationBetweenProjects(t *testing.T) {
	tmpDir := t.TempDir()

	// Create two projects with separate venvs
	projectA := filepath.Join(tmpDir, "project-a")
	projectB := filepath.Join(tmpDir, "project-b")

	for _, proj := range []string{projectA, projectB} {
		venvPath := filepath.Join(proj, ".venv")
		binDir := filepath.Join(venvPath, testBinDir())
		os.MkdirAll(binDir, 0755)
		os.WriteFile(filepath.Join(venvPath, "pyvenv.cfg"), []byte("home = /usr/bin\n"), 0644)
		os.WriteFile(filepath.Join(binDir, testPythonBin()), []byte("#!/bin/sh\n"), 0755)
	}

	// Detect from project-a finds project-a's venv
	vA, err := Detect(projectA)
	if err != nil {
		t.Fatalf("should find project-a venv: %v", err)
	}
	expectedA := filepath.Join(projectA, ".venv")
	if vA.Path != expectedA {
		t.Errorf("project-a: expected %q, got %q", expectedA, vA.Path)
	}

	// Detect from project-b finds project-b's venv
	vB, err := Detect(projectB)
	if err != nil {
		t.Fatalf("should find project-b venv: %v", err)
	}
	expectedB := filepath.Join(projectB, ".venv")
	if vB.Path != expectedB {
		t.Errorf("project-b: expected %q, got %q", expectedB, vB.Path)
	}

	// They must be different
	if vA.Path == vB.Path {
		t.Error("two projects should have different venvs")
	}
}

func TestSitePackagesDir(t *testing.T) {
	tmpDir := t.TempDir()
	venvPath := filepath.Join(tmpDir, ".venv")

	var spDir string
	if runtime.GOOS == "windows" {
		spDir = filepath.Join(venvPath, "Lib", "site-packages")
	} else {
		spDir = filepath.Join(venvPath, "lib", "python3.12", "site-packages")
	}
	os.MkdirAll(spDir, 0755)

	result := findSitePackages(venvPath)
	if result != spDir {
		t.Errorf("expected %q, got %q", spDir, result)
	}
}

func TestFindSitePackages_EmptyLib(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Windows always finds Lib/site-packages if Lib exists")
	}
	tmpDir := t.TempDir()
	venvPath := filepath.Join(tmpDir, ".venv")
	os.MkdirAll(filepath.Join(venvPath, "lib"), 0755)

	result := findSitePackages(venvPath)
	if result != "" {
		t.Errorf("expected empty string for venv with no python dir, got %q", result)
	}
}

func TestDetect_IgnoresVIRTUAL_ENVFromOutside(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a venv at a path that is NOT reachable by walking up from cwd
	venvPath := filepath.Join(tmpDir, "other-project", ".venv")
	binDir := filepath.Join(venvPath, testBinDir())
	os.MkdirAll(binDir, 0755)
	os.WriteFile(filepath.Join(venvPath, "pyvenv.cfg"), []byte("home = /usr/bin\n"), 0644)
	os.WriteFile(filepath.Join(binDir, testPythonBin()), []byte("#!/bin/sh\n"), 0755)

	// Set VIRTUAL_ENV to point to it
	t.Setenv("VIRTUAL_ENV", venvPath)

	// Detect from a directory with NO local venv should NOT use VIRTUAL_ENV
	noVenvDir := filepath.Join(tmpDir, "no-venv-here")
	os.MkdirAll(noVenvDir, 0755)

	_, err := Detect(noVenvDir)
	if err == nil {
		t.Error("Detect should NOT find venv via VIRTUAL_ENV when outside the project directory")
	}
}

func TestDetect_LocalVenvWins_OverVIRTUAL_ENV(t *testing.T) {
	tmpDir := t.TempDir()

	// Create local .venv
	localVenv := filepath.Join(tmpDir, ".venv")
	localBin := filepath.Join(localVenv, testBinDir())
	os.MkdirAll(localBin, 0755)
	os.WriteFile(filepath.Join(localVenv, "pyvenv.cfg"), []byte("home = /usr/bin\n"), 0644)
	os.WriteFile(filepath.Join(localBin, testPythonBin()), []byte("#!/bin/sh\n"), 0755)

	// Create remote venv referenced by VIRTUAL_ENV
	remoteVenv := filepath.Join(tmpDir, "remote", ".venv")
	remoteBin := filepath.Join(remoteVenv, testBinDir())
	os.MkdirAll(remoteBin, 0755)
	os.WriteFile(filepath.Join(remoteVenv, "pyvenv.cfg"), []byte("home = /usr/bin\n"), 0644)
	os.WriteFile(filepath.Join(remoteBin, testPythonBin()), []byte("#!/bin/sh\n"), 0755)

	// VIRTUAL_ENV points to remote venv — should be IGNORED
	t.Setenv("VIRTUAL_ENV", remoteVenv)

	v, err := Detect(tmpDir)
	if err != nil {
		t.Fatalf("Detect should succeed via local .venv: %v", err)
	}
	// Local directory detection wins — VIRTUAL_ENV is ignored
	if v.Path != localVenv {
		t.Errorf("local .venv should win: expected %q, got %q", localVenv, v.Path)
	}
}

func TestDetect_IgnoresVIRTUAL_ENV_UsesLocalVenv(t *testing.T) {
	tmpDir := t.TempDir()

	// Create local .venv
	localVenv := filepath.Join(tmpDir, ".venv")
	localBin := filepath.Join(localVenv, testBinDir())
	os.MkdirAll(localBin, 0755)
	os.WriteFile(filepath.Join(localVenv, "pyvenv.cfg"), []byte("home = /usr/bin\n"), 0644)
	os.WriteFile(filepath.Join(localBin, testPythonBin()), []byte("#!/bin/sh\n"), 0755)

	// VIRTUAL_ENV is set but fpm should ignore it entirely
	t.Setenv("VIRTUAL_ENV", "/some/other/path/.venv")

	// Should detect local .venv by directory only
	v, err := Detect(tmpDir)
	if err != nil {
		t.Fatalf("Detect should find local venv by directory: %v", err)
	}
	if v.Path != localVenv {
		t.Errorf("expected local %q, got %q", localVenv, v.Path)
	}
}

func TestDetect_UnsetVIRTUAL_ENV_UsesDirectory(t *testing.T) {
	tmpDir := t.TempDir()

	// Create local .venv
	localVenv := filepath.Join(tmpDir, ".venv")
	localBin := filepath.Join(localVenv, testBinDir())
	os.MkdirAll(localBin, 0755)
	os.WriteFile(filepath.Join(localVenv, "pyvenv.cfg"), []byte("home = /usr/bin\n"), 0644)
	os.WriteFile(filepath.Join(localBin, testPythonBin()), []byte("#!/bin/sh\n"), 0755)

	// No VIRTUAL_ENV set (simulates "deactivate" was called)
	t.Setenv("VIRTUAL_ENV", "")

	// Should detect local .venv by directory
	v, err := Detect(tmpDir)
	if err != nil {
		t.Fatalf("Detect should find local .venv: %v", err)
	}
	if v.Path != localVenv {
		t.Errorf("expected local %q, got %q", localVenv, v.Path)
	}
}

func TestFindSitePackages_MultiplePythonVersions(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Windows uses Lib/site-packages without python version dirs")
	}
	tmpDir := t.TempDir()
	venvPath := filepath.Join(tmpDir, ".venv")

	// Create multiple python version dirs (only one has site-packages)
	os.MkdirAll(filepath.Join(venvPath, "lib", "python3.11"), 0755)
	sp312 := filepath.Join(venvPath, "lib", "python3.12", "site-packages")
	os.MkdirAll(sp312, 0755)

	result := findSitePackages(venvPath)
	// Should find the one with site-packages
	if result != sp312 {
		t.Errorf("expected %q, got %q", sp312, result)
	}
}
