package depgraph

import (
	"os"
	"path/filepath"
	"sort"
	"testing"
)

func TestOrphans_Basic(t *testing.T) {
	g := &Graph{Packages: make(map[string]*PackageNode)}

	// flask (requested) depends on werkzeug, jinja2, click
	g.AddRequested("flask", "3.0.0", []string{"werkzeug", "jinja2", "click"})
	g.AddTransitive("werkzeug", "3.0.0", nil)
	g.AddTransitive("jinja2", "3.0.0", []string{"markupsafe"})
	g.AddTransitive("click", "8.0.0", nil)
	g.AddTransitive("markupsafe", "2.1.0", nil)

	orphans := g.Orphans()
	if len(orphans) != 0 {
		t.Errorf("expected 0 orphans while flask is installed, got %v", orphans)
	}
}

func TestOrphans_AfterRemoveRequested(t *testing.T) {
	g := &Graph{Packages: make(map[string]*PackageNode)}

	g.AddRequested("flask", "3.0.0", []string{"werkzeug", "jinja2", "click"})
	g.AddTransitive("werkzeug", "3.0.0", []string{"markupsafe"})
	g.AddTransitive("jinja2", "3.0.0", []string{"markupsafe"})
	g.AddTransitive("click", "8.0.0", nil)
	g.AddTransitive("markupsafe", "2.1.0", nil)

	// Remove flask
	g.Remove("flask")

	// First pass: werkzeug, jinja2, click are orphans (not needed by anyone)
	// markupsafe is still needed by werkzeug and jinja2
	orphans := g.Orphans()
	sort.Strings(orphans)

	expected := []string{"click", "jinja2", "werkzeug"}
	sort.Strings(expected)

	if len(orphans) != len(expected) {
		t.Fatalf("expected orphans %v, got %v", expected, orphans)
	}
	for i, name := range expected {
		if orphans[i] != name {
			t.Errorf("expected orphan[%d] = %q, got %q", i, name, orphans[i])
		}
	}
}

func TestOrphans_RecursiveAfterFirstLayerRemoved(t *testing.T) {
	g := &Graph{Packages: make(map[string]*PackageNode)}

	// Flask depends on werkzeug, jinja2, click
	// werkzeug depends on markupsafe
	// jinja2 depends on markupsafe
	// importlib-metadata depends on zipp
	g.AddRequested("flask", "3.0.0", []string{"werkzeug", "jinja2", "click", "importlib-metadata"})
	g.AddTransitive("werkzeug", "3.0.0", []string{"markupsafe"})
	g.AddTransitive("jinja2", "3.0.0", []string{"markupsafe"})
	g.AddTransitive("click", "8.0.0", nil)
	g.AddTransitive("importlib-metadata", "9.0.0", []string{"zipp"})
	g.AddTransitive("markupsafe", "2.1.0", nil)
	g.AddTransitive("zipp", "4.0.0", nil)

	// Remove flask
	g.Remove("flask")

	// First-layer orphans: werkzeug, jinja2, click, importlib-metadata
	orphans := g.Orphans()
	sort.Strings(orphans)
	expectedFirst := []string{"click", "importlib-metadata", "jinja2", "werkzeug"}
	sort.Strings(expectedFirst)

	if len(orphans) != len(expectedFirst) {
		t.Fatalf("first pass: expected orphans %v, got %v", expectedFirst, orphans)
	}

	// Remove first-layer orphans
	for _, name := range orphans {
		g.Remove(name)
	}

	// Second-layer orphans: markupsafe and zipp should now be orphaned
	orphans2 := g.Orphans()
	sort.Strings(orphans2)
	expectedSecond := []string{"markupsafe", "zipp"}
	sort.Strings(expectedSecond)

	if len(orphans2) != len(expectedSecond) {
		t.Fatalf("second pass: expected orphans %v, got %v", expectedSecond, orphans2)
	}
	for i, name := range expectedSecond {
		if orphans2[i] != name {
			t.Errorf("second pass: expected orphan[%d] = %q, got %q", i, name, orphans2[i])
		}
	}

	// Remove second-layer orphans
	for _, name := range orphans2 {
		g.Remove(name)
	}

	// Third pass: nothing left
	orphans3 := g.Orphans()
	if len(orphans3) != 0 {
		t.Errorf("third pass: expected 0 orphans, got %v", orphans3)
	}
}

func TestOrphans_SharedDependencyNotOrphaned(t *testing.T) {
	g := &Graph{Packages: make(map[string]*PackageNode)}

	// Both flask and requests depend on urllib3
	g.AddRequested("flask", "3.0.0", []string{"werkzeug"})
	g.AddRequested("requests", "2.31.0", []string{"urllib3", "charset-normalizer"})
	g.AddTransitive("werkzeug", "3.0.0", nil)
	g.AddTransitive("urllib3", "2.1.0", nil)
	g.AddTransitive("charset-normalizer", "3.0.0", nil)

	// Remove flask — werkzeug is orphaned but urllib3 is NOT (requests needs it)
	g.Remove("flask")

	orphans := g.Orphans()
	if len(orphans) != 1 || orphans[0] != "werkzeug" {
		t.Errorf("expected only werkzeug as orphan, got %v", orphans)
	}
}

func TestOrphans_RequestedNeverOrphaned(t *testing.T) {
	g := &Graph{Packages: make(map[string]*PackageNode)}

	// User explicitly installed both flask and jinja2
	g.AddRequested("flask", "3.0.0", []string{"jinja2", "werkzeug"})
	g.AddRequested("jinja2", "3.0.0", []string{"markupsafe"})
	g.AddTransitive("werkzeug", "3.0.0", nil)
	g.AddTransitive("markupsafe", "2.1.0", nil)

	// Remove flask
	g.Remove("flask")

	// jinja2 is NOT an orphan because it's requested
	// werkzeug IS an orphan
	// markupsafe is NOT an orphan because jinja2 still needs it
	orphans := g.Orphans()
	if len(orphans) != 1 || orphans[0] != "werkzeug" {
		t.Errorf("expected only werkzeug as orphan (jinja2 is requested), got %v", orphans)
	}
}

func TestOrphans_DiamondDependency(t *testing.T) {
	g := &Graph{Packages: make(map[string]*PackageNode)}

	// Diamond: flask → {jinja2, werkzeug} → markupsafe
	g.AddRequested("flask", "3.0.0", []string{"jinja2", "werkzeug"})
	g.AddTransitive("jinja2", "3.0.0", []string{"markupsafe"})
	g.AddTransitive("werkzeug", "3.0.0", []string{"markupsafe"})
	g.AddTransitive("markupsafe", "2.1.0", nil)

	g.Remove("flask")

	// First layer: jinja2 and werkzeug
	orphans := g.Orphans()
	sort.Strings(orphans)
	if len(orphans) != 2 {
		t.Fatalf("expected 2 orphans, got %v", orphans)
	}

	// Remove both — markupsafe should now be orphaned
	g.Remove("jinja2")
	g.Remove("werkzeug")

	orphans2 := g.Orphans()
	if len(orphans2) != 1 || orphans2[0] != "markupsafe" {
		t.Errorf("expected markupsafe orphaned after removing both parents, got %v", orphans2)
	}
}

func TestOrphans_PartialDiamondRemoval(t *testing.T) {
	g := &Graph{Packages: make(map[string]*PackageNode)}

	// Diamond: flask → {jinja2, werkzeug} → markupsafe
	g.AddRequested("flask", "3.0.0", []string{"jinja2", "werkzeug"})
	g.AddTransitive("jinja2", "3.0.0", []string{"markupsafe"})
	g.AddTransitive("werkzeug", "3.0.0", []string{"markupsafe"})
	g.AddTransitive("markupsafe", "2.1.0", nil)

	g.Remove("flask")

	// Only remove jinja2 (user chose to keep werkzeug)
	g.Remove("jinja2")

	// markupsafe is NOT orphaned because werkzeug still needs it
	orphans := g.Orphans()
	if len(orphans) != 1 || orphans[0] != "werkzeug" {
		t.Errorf("expected werkzeug as only remaining orphan (markupsafe still needed by werkzeug which is still present), got %v", orphans)
	}
}

func TestGraphSaveAndLoad(t *testing.T) {
	tmpDir := t.TempDir()
	envPath := tmpDir

	g := &Graph{Packages: make(map[string]*PackageNode)}
	g.AddRequested("requests", "2.31.0", []string{"urllib3"})
	g.AddTransitive("urllib3", "2.1.0", nil)

	if err := g.Save(envPath); err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	// Verify file exists
	path := filepath.Join(envPath, ".fpm-depgraph.json")
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("graph file not created: %v", err)
	}

	// Load and verify
	loaded := Load(envPath)
	if len(loaded.Packages) != 2 {
		t.Fatalf("expected 2 packages, got %d", len(loaded.Packages))
	}
	if !loaded.IsRequested("requests") {
		t.Error("requests should be requested")
	}
	if loaded.IsRequested("urllib3") {
		t.Error("urllib3 should NOT be requested")
	}
}

func TestGraphIsolation_SeparateEnvPaths(t *testing.T) {
	// Simulate two separate environments (venv vs global)
	venvDir := t.TempDir()
	globalDir := t.TempDir()

	// Add packages to venv graph
	venvGraph := &Graph{Packages: make(map[string]*PackageNode)}
	venvGraph.AddRequested("flask", "3.0.0", []string{"werkzeug"})
	venvGraph.AddTransitive("werkzeug", "3.0.0", nil)
	venvGraph.Save(venvDir)

	// Add different packages to global graph
	globalGraph := &Graph{Packages: make(map[string]*PackageNode)}
	globalGraph.AddRequested("requests", "2.31.0", []string{"urllib3"})
	globalGraph.AddTransitive("urllib3", "2.1.0", nil)
	globalGraph.Save(globalDir)

	// Load venv graph — should only have flask
	loadedVenv := Load(venvDir)
	if _, ok := loadedVenv.Packages["flask"]; !ok {
		t.Error("venv graph should have flask")
	}
	if _, ok := loadedVenv.Packages["requests"]; ok {
		t.Error("venv graph should NOT have requests (that's in global)")
	}

	// Load global graph — should only have requests
	loadedGlobal := Load(globalDir)
	if _, ok := loadedGlobal.Packages["requests"]; !ok {
		t.Error("global graph should have requests")
	}
	if _, ok := loadedGlobal.Packages["flask"]; ok {
		t.Error("global graph should NOT have flask (that's in venv)")
	}
}

func TestDependentsOf(t *testing.T) {
	g := &Graph{Packages: make(map[string]*PackageNode)}
	g.AddRequested("flask", "3.0.0", []string{"jinja2", "werkzeug"})
	g.AddTransitive("jinja2", "3.0.0", []string{"markupsafe"})
	g.AddTransitive("werkzeug", "3.0.0", []string{"markupsafe"})
	g.AddTransitive("markupsafe", "2.1.0", nil)

	// markupsafe is depended on by jinja2 and werkzeug
	deps := g.DependentsOf("markupsafe")
	sort.Strings(deps)
	if len(deps) != 2 {
		t.Fatalf("expected 2 dependents of markupsafe, got %v", deps)
	}
	if deps[0] != "jinja2" || deps[1] != "werkzeug" {
		t.Errorf("expected [jinja2, werkzeug], got %v", deps)
	}

	// flask has no dependents (it's a root)
	deps = g.DependentsOf("flask")
	if len(deps) != 0 {
		t.Errorf("expected 0 dependents of flask, got %v", deps)
	}
}

func TestAddTransitive_DoesNotDowngradeRequested(t *testing.T) {
	g := &Graph{Packages: make(map[string]*PackageNode)}

	// User explicitly installs jinja2
	g.AddRequested("jinja2", "3.0.0", []string{"markupsafe"})

	// Later, flask is installed and tries to add jinja2 as transitive
	g.AddTransitive("jinja2", "3.1.0", []string{"markupsafe"})

	// jinja2 should still be requested (not downgraded to transitive)
	if !g.IsRequested("jinja2") {
		t.Error("jinja2 should remain requested after AddTransitive")
	}
	// Version should be updated
	if g.Packages["jinja2"].Version != "3.1.0" {
		t.Errorf("version should be updated to 3.1.0, got %s", g.Packages["jinja2"].Version)
	}
}
