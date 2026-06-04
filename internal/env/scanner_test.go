package env

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/kartikeyyadav/fpm/internal/pep440"
	"github.com/kartikeyyadav/fpm/pkg/types"
)

func TestScanSitePackages(t *testing.T) {
	// Create a temporary site-packages directory with mock dist-info
	tmpDir := t.TempDir()

	// Create numpy-1.24.0.dist-info
	distInfo := filepath.Join(tmpDir, "numpy-1.24.0.dist-info")
	os.MkdirAll(distInfo, 0755)
	os.WriteFile(filepath.Join(distInfo, "METADATA"), []byte(
		"Metadata-Version: 2.1\nName: numpy\nVersion: 1.24.0\n\nNumPy is...\n",
	), 0644)
	os.WriteFile(filepath.Join(distInfo, "INSTALLER"), []byte("pip\n"), 0644)

	// Create requests-2.31.0.dist-info
	distInfo2 := filepath.Join(tmpDir, "requests-2.31.0.dist-info")
	os.MkdirAll(distInfo2, 0755)
	os.WriteFile(filepath.Join(distInfo2, "METADATA"), []byte(
		"Metadata-Version: 2.1\nName: requests\nVersion: 2.31.0\n\n",
	), 0644)
	os.WriteFile(filepath.Join(distInfo2, "INSTALLER"), []byte("fpm\n"), 0644)

	scanner := NewScanner([]string{tmpDir})
	result, err := scanner.Scan()
	if err != nil {
		t.Fatalf("Scan() error: %v", err)
	}

	if len(result.Packages) != 2 {
		t.Fatalf("expected 2 packages, got %d", len(result.Packages))
	}

	// Check numpy
	numpy := result.FindByName(types.NewPackageName("numpy"))
	if len(numpy) != 1 {
		t.Fatalf("expected 1 numpy, got %d", len(numpy))
	}
	if numpy[0].Manager != ManagerPip {
		t.Errorf("numpy manager = %s, want pip", numpy[0].Manager)
	}
	expectedVer, _ := pep440.Parse("1.24.0")
	if !numpy[0].Version.Equal(expectedVer) {
		t.Errorf("numpy version = %s, want 1.24.0", numpy[0].Version)
	}

	// Check requests
	reqs := result.FindByName(types.NewPackageName("requests"))
	if len(reqs) != 1 {
		t.Fatalf("expected 1 requests, got %d", len(reqs))
	}
	if reqs[0].Manager != ManagerFpm {
		t.Errorf("requests manager = %s, want fpm", reqs[0].Manager)
	}
}

func TestCrossManagerSameVersion(t *testing.T) {
	scan := &ScanResult{
		Packages: []InstalledPackage{
			{
				Name:    types.NewPackageName("numpy"),
				Version: mustParse("1.24.0"),
				Manager: ManagerPip,
			},
		},
	}

	checker := NewCrossManagerChecker(scan, PolicyAsk)
	result := checker.Check(types.NewPackageName("numpy"), mustParse("1.24.0"))

	if result.Action != ActionSkip {
		t.Errorf("expected ActionSkip, got %d", result.Action)
	}
	if result.Message == "" {
		t.Error("expected a message")
	}
}

func TestCrossManagerDifferentVersionInstallPolicy(t *testing.T) {
	scan := &ScanResult{
		Packages: []InstalledPackage{
			{
				Name:    types.NewPackageName("numpy"),
				Version: mustParse("1.23.0"),
				Manager: ManagerPip,
			},
		},
	}

	checker := NewCrossManagerChecker(scan, PolicyInstall)
	result := checker.Check(types.NewPackageName("numpy"), mustParse("1.24.0"))

	if result.Action != ActionInstall {
		t.Errorf("expected ActionInstall, got %d", result.Action)
	}
}

func TestCrossManagerDifferentVersionSkipPolicy(t *testing.T) {
	scan := &ScanResult{
		Packages: []InstalledPackage{
			{
				Name:    types.NewPackageName("numpy"),
				Version: mustParse("1.23.0"),
				Manager: ManagerPip,
			},
		},
	}

	checker := NewCrossManagerChecker(scan, PolicySkip)
	result := checker.Check(types.NewPackageName("numpy"), mustParse("1.24.0"))

	if result.Action != ActionSkip {
		t.Errorf("expected ActionSkip, got %d", result.Action)
	}
}

func TestCrossManagerNoExisting(t *testing.T) {
	scan := &ScanResult{Packages: nil}

	checker := NewCrossManagerChecker(scan, PolicyAsk)
	result := checker.Check(types.NewPackageName("scipy"), mustParse("1.11.0"))

	if result.Action != ActionInstall {
		t.Errorf("expected ActionInstall, got %d", result.Action)
	}
}

func TestPthFileGeneration(t *testing.T) {
	tmpDir := t.TempDir()

	cfg := PthConfig{
		Layers: []PthLayer{
			{Name: "fpm-local", Path: "/home/user/project/.venv/lib/python3.11/site-packages"},
			{Name: "fpm-global", Path: "/home/user/.local/share/fpm/global/site-packages"},
		},
	}

	err := GeneratePthFile(cfg, tmpDir)
	if err != nil {
		t.Fatalf("GeneratePthFile error: %v", err)
	}

	// Verify file exists
	pthPath := filepath.Join(tmpDir, pthFileName)
	data, err := os.ReadFile(pthPath)
	if err != nil {
		t.Fatalf("ReadFile error: %v", err)
	}

	content := string(data)
	if !contains(content, "fpm-local") {
		t.Error("expected fpm-local layer in .pth file")
	}
	if !contains(content, "fpm-global") {
		t.Error("expected fpm-global layer in .pth file")
	}

	// Read it back
	readCfg, err := ReadPthFile(tmpDir)
	if err != nil {
		t.Fatalf("ReadPthFile error: %v", err)
	}
	if len(readCfg.Layers) != 2 {
		t.Errorf("expected 2 layers, got %d", len(readCfg.Layers))
	}
}

func mustParse(s string) pep440.Version {
	v, err := pep440.Parse(s)
	if err != nil {
		panic(err)
	}
	return v
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsStr(s, substr))
}

func containsStr(s, sub string) bool {
	for i := 0; i <= len(s)-len(sub); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
