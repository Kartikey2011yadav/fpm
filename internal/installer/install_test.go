package installer

import (
	"archive/zip"
	"os"
	"path/filepath"
	"testing"

	"github.com/kartikeyyadav/fpm/internal/cache"
	"github.com/kartikeyyadav/fpm/internal/fs"
	"github.com/kartikeyyadav/fpm/pkg/types"
)

func TestInstallFromCAS(t *testing.T) {
	tmpDir := t.TempDir()
	cacheDir := filepath.Join(tmpDir, "cache")
	sitePackages := filepath.Join(tmpDir, "site-packages")
	os.MkdirAll(sitePackages, 0755)

	c := cache.New(cacheDir)
	c.Init()

	// Create a mock wheel and store in CAS
	wheelPath := createTestWheel(t, tmpDir, "testpkg", "1.0.0")
	casKey, err := c.Store(wheelPath)
	if err != nil {
		t.Fatalf("Store failed: %v", err)
	}

	// Install
	plan := InstallPlan{
		Package:   types.NewPackageName("testpkg"),
		Version:   "1.0.0",
		CASKey:    casKey,
		TargetDir: sitePackages,
		LinkMode:  fs.LinkModeCopy,
	}

	result, err := Install(plan, c)
	if err != nil {
		t.Fatalf("Install failed: %v", err)
	}

	if result.InstalledFiles == 0 {
		t.Error("expected files to be installed")
	}

	// Verify files exist
	initFile := filepath.Join(sitePackages, "testpkg", "__init__.py")
	if _, err := os.Stat(initFile); err != nil {
		t.Errorf("installed file not found: %s", initFile)
	}

	// Verify INSTALLER marker
	if result.DistInfoDir != "" {
		installerFile := filepath.Join(result.DistInfoDir, "INSTALLER")
		data, err := os.ReadFile(installerFile)
		if err != nil {
			t.Errorf("INSTALLER file not found")
		} else if string(data) != "fpm\n" {
			t.Errorf("INSTALLER = %q, want 'fpm\\n'", data)
		}
	}
}

func TestUninstall(t *testing.T) {
	tmpDir := t.TempDir()
	sitePackages := filepath.Join(tmpDir, "site-packages")
	os.MkdirAll(sitePackages, 0755)

	// Create a fake installed package
	pkgDir := filepath.Join(sitePackages, "testpkg")
	os.MkdirAll(pkgDir, 0755)
	os.WriteFile(filepath.Join(pkgDir, "__init__.py"), []byte(""), 0644)

	distInfo := filepath.Join(sitePackages, "testpkg-1.0.0.dist-info")
	os.MkdirAll(distInfo, 0755)
	os.WriteFile(filepath.Join(distInfo, "METADATA"), []byte("Name: testpkg\nVersion: 1.0.0\n"), 0644)
	os.WriteFile(filepath.Join(distInfo, "INSTALLER"), []byte("fpm\n"), 0644)
	os.WriteFile(filepath.Join(distInfo, "RECORD"), []byte(
		"testpkg/__init__.py,sha256=abc,0\ntestpkg-1.0.0.dist-info/METADATA,,\ntestpkg-1.0.0.dist-info/RECORD,,\n",
	), 0644)

	// Uninstall
	result, err := Uninstall(types.NewPackageName("testpkg"), sitePackages)
	if err != nil {
		t.Fatalf("Uninstall failed: %v", err)
	}

	if result.RemovedFiles == 0 {
		t.Error("expected files to be removed")
	}

	// Verify files removed
	if _, err := os.Stat(filepath.Join(pkgDir, "__init__.py")); !os.IsNotExist(err) {
		t.Error("__init__.py should be removed")
	}
	if _, err := os.Stat(distInfo); !os.IsNotExist(err) {
		t.Error("dist-info should be removed")
	}
}

func TestEntryPointParsing(t *testing.T) {
	tmpDir := t.TempDir()
	entryPointsFile := filepath.Join(tmpDir, "entry_points.txt")
	os.WriteFile(entryPointsFile, []byte(`[console_scripts]
black = black:patched_main
blackd = blackd:main
`), 0644)

	scripts, err := parseEntryPoints(entryPointsFile)
	if err != nil {
		t.Fatalf("parseEntryPoints failed: %v", err)
	}

	if len(scripts) != 2 {
		t.Fatalf("expected 2 scripts, got %d", len(scripts))
	}
	if scripts[0].Name != "black" {
		t.Errorf("scripts[0].Name = %q, want 'black'", scripts[0].Name)
	}
	if scripts[0].Module != "black" {
		t.Errorf("scripts[0].Module = %q, want 'black'", scripts[0].Module)
	}
	if scripts[0].Func != "patched_main" {
		t.Errorf("scripts[0].Func = %q, want 'patched_main'", scripts[0].Func)
	}
}

func createTestWheel(t *testing.T, dir, name, version string) string {
	t.Helper()
	wheelPath := filepath.Join(dir, name+"-"+version+"-py3-none-any.whl")
	f, _ := os.Create(wheelPath)
	w := zip.NewWriter(f)
	fw, _ := w.Create(name + "/__init__.py")
	fw.Write([]byte("__version__ = \"" + version + "\"\n"))
	fw, _ = w.Create(name + "-" + version + ".dist-info/METADATA")
	fw.Write([]byte("Metadata-Version: 2.1\nName: " + name + "\nVersion: " + version + "\n"))
	fw, _ = w.Create(name + "-" + version + ".dist-info/RECORD")
	fw.Write([]byte(name + "/__init__.py,,\n"))
	w.Close()
	f.Close()
	return wheelPath
}
