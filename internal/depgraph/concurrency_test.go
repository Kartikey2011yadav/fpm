package depgraph

import (
	"path/filepath"
	"sync"
	"testing"
)

// Tests that simulate git-like distributed conflicts in the dependency graph.
// These cover multi-user scenarios, race conditions, and version conflicts.

func TestConcurrent_SaveLoadRace(t *testing.T) {
	tmpDir := t.TempDir()

	// Simulate two users modifying the same environment graph simultaneously
	var wg sync.WaitGroup
	for i := 0; i < 20; i++ {
		wg.Add(2)
		go func(n int) {
			defer wg.Done()
			g := Load(tmpDir)
			g.AddRequested("user1-pkg-"+string(rune('a'+n%26)), "1.0.0", nil)
			g.Save(tmpDir)
		}(i)
		go func(n int) {
			defer wg.Done()
			g := Load(tmpDir)
			g.AddRequested("user2-pkg-"+string(rune('a'+n%26)), "2.0.0", nil)
			g.Save(tmpDir)
		}(i)
	}
	wg.Wait()

	// Verify the graph is valid JSON (not corrupted)
	final := Load(tmpDir)
	if final.Packages == nil {
		t.Fatal("graph corrupted: nil packages map")
	}
	// Due to race conditions, not all packages may be present,
	// but the file should never be corrupted/unparseable
	if len(final.Packages) == 0 {
		t.Error("expected at least some packages in graph after concurrent writes")
	}
}

func TestConcurrent_IsolatedEnvsSafe(t *testing.T) {
	// Two separate environments should never interfere
	env1 := t.TempDir()
	env2 := t.TempDir()

	var wg sync.WaitGroup
	for i := 0; i < 50; i++ {
		wg.Add(2)
		go func(n int) {
			defer wg.Done()
			g := Load(env1)
			g.AddRequested("env1-pkg", "1.0.0", []string{"dep-a"})
			g.AddTransitive("dep-a", "1.0.0", nil)
			g.Save(env1)
		}(i)
		go func(n int) {
			defer wg.Done()
			g := Load(env2)
			g.AddRequested("env2-pkg", "2.0.0", []string{"dep-b"})
			g.AddTransitive("dep-b", "2.0.0", nil)
			g.Save(env2)
		}(i)
	}
	wg.Wait()

	g1 := Load(env1)
	g2 := Load(env2)

	if _, ok := g1.Packages["env2-pkg"]; ok {
		t.Error("env1 graph should NOT contain env2 packages")
	}
	if _, ok := g2.Packages["env1-pkg"]; ok {
		t.Error("env2 graph should NOT contain env1 packages")
	}
}

func TestConcurrent_OrphanDuringRemove(t *testing.T) {
	tmpDir := t.TempDir()

	// Setup: A depends on B, C depends on B
	g := &Graph{Packages: make(map[string]*PackageNode)}
	g.AddRequested("a", "1.0", []string{"b"})
	g.AddRequested("c", "1.0", []string{"b"})
	g.AddTransitive("b", "1.0", nil)
	g.Save(tmpDir)

	// Simulate two users removing different packages simultaneously
	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		defer wg.Done()
		g := Load(tmpDir)
		g.Remove("a")
		g.Save(tmpDir)
	}()

	go func() {
		defer wg.Done()
		g := Load(tmpDir)
		g.Remove("c")
		g.Save(tmpDir)
	}()

	wg.Wait()

	// After both removes, b should be orphaned
	final := Load(tmpDir)
	// Due to race, one or both removes may have succeeded
	// but the file should be valid
	if final.Packages == nil {
		t.Fatal("graph corrupted after concurrent removes")
	}
}

func TestConflict_VersionMismatch(t *testing.T) {
	// Simulate: User A installs pkg==1.0, User B installs pkg==2.0
	// The last write wins (like git push --force)
	tmpDir := t.TempDir()

	g1 := &Graph{Packages: make(map[string]*PackageNode)}
	g1.AddRequested("numpy", "1.24.0", nil)
	g1.Save(tmpDir)

	g2 := Load(tmpDir)
	g2.AddRequested("numpy", "2.0.0", nil)
	g2.Save(tmpDir)

	final := Load(tmpDir)
	if final.Packages["numpy"].Version != "2.0.0" {
		t.Errorf("last write should win: expected 2.0.0, got %s", final.Packages["numpy"].Version)
	}
}

func TestConflict_SimultaneousAddRemove(t *testing.T) {
	// User A adds a package, User B removes the same package
	tmpDir := t.TempDir()

	g := &Graph{Packages: make(map[string]*PackageNode)}
	g.AddRequested("flask", "3.0.0", []string{"werkzeug"})
	g.AddTransitive("werkzeug", "3.0.0", nil)
	g.Save(tmpDir)

	var wg sync.WaitGroup
	wg.Add(2)

	// User A: upgrade flask
	go func() {
		defer wg.Done()
		g := Load(tmpDir)
		g.AddRequested("flask", "3.1.0", []string{"werkzeug"})
		g.Save(tmpDir)
	}()

	// User B: remove flask
	go func() {
		defer wg.Done()
		g := Load(tmpDir)
		g.Remove("flask")
		g.Save(tmpDir)
	}()

	wg.Wait()

	// One of the two states should be valid — never corrupt
	final := Load(tmpDir)
	if final.Packages == nil {
		t.Fatal("graph corrupted after add/remove race")
	}
}

func TestConflict_SharedDepRemovalRace(t *testing.T) {
	// User A removes pkg1 (shares dep X with pkg2)
	// User B removes pkg2 (shares dep X with pkg1)
	// Both check orphans — dep X should become orphan after BOTH are gone
	tmpDir := t.TempDir()

	g := &Graph{Packages: make(map[string]*PackageNode)}
	g.AddRequested("pkg1", "1.0", []string{"shared-dep"})
	g.AddRequested("pkg2", "1.0", []string{"shared-dep"})
	g.AddTransitive("shared-dep", "1.0", nil)
	g.Save(tmpDir)

	// Sequential to verify logic (concurrent version may produce either state)
	g1 := Load(tmpDir)
	g1.Remove("pkg1")
	orphans1 := g1.Orphans()
	if len(orphans1) != 0 {
		t.Error("shared-dep should NOT be orphaned after removing only pkg1")
	}
	g1.Save(tmpDir)

	g2 := Load(tmpDir)
	g2.Remove("pkg2")
	orphans2 := g2.Orphans()
	if len(orphans2) != 1 || orphans2[0] != "shared-dep" {
		t.Errorf("shared-dep should be orphaned after removing both, got: %v", orphans2)
	}
}

func TestMultiUser_SeparateGraphFiles(t *testing.T) {
	// Verify that per-environment graphs don't interfere
	base := t.TempDir()
	env1 := filepath.Join(base, "project-a", ".venv")
	env2 := filepath.Join(base, "project-b", ".venv")

	g1 := &Graph{Packages: make(map[string]*PackageNode)}
	g1.AddRequested("flask", "3.0.0", nil)
	g1.Save(env1)

	g2 := &Graph{Packages: make(map[string]*PackageNode)}
	g2.AddRequested("django", "5.0.0", nil)
	g2.Save(env2)

	// Verify isolation
	loaded1 := Load(env1)
	loaded2 := Load(env2)

	if _, ok := loaded1.Packages["django"]; ok {
		t.Error("project-a should not see project-b's packages")
	}
	if _, ok := loaded2.Packages["flask"]; ok {
		t.Error("project-b should not see project-a's packages")
	}
}

func TestMultiUser_ConcurrentOrphanComputation(t *testing.T) {
	g := &Graph{Packages: make(map[string]*PackageNode)}
	g.AddRequested("root", "1.0", []string{"dep1", "dep2", "dep3"})
	g.AddTransitive("dep1", "1.0", nil)
	g.AddTransitive("dep2", "1.0", nil)
	g.AddTransitive("dep3", "1.0", nil)

	// Multiple goroutines calling Orphans concurrently should be safe
	var wg sync.WaitGroup
	results := make([][]string, 100)
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			results[idx] = g.Orphans()
		}(i)
	}
	wg.Wait()

	// All should return the same result (0 orphans while root exists)
	for i, r := range results {
		if len(r) != 0 {
			t.Errorf("goroutine %d: expected 0 orphans, got %v", i, r)
		}
	}
}

func TestMultiUser_MarkWhileAutoremove(t *testing.T) {
	// Simulate: User A marks a dep as requested while User B autoremoves
	g := &Graph{Packages: make(map[string]*PackageNode)}
	g.AddRequested("flask", "3.0.0", []string{"click", "werkzeug"})
	g.AddTransitive("click", "8.0.0", nil)
	g.AddTransitive("werkzeug", "3.0.0", nil)

	g.Remove("flask")

	// User A marks click as requested (protecting it)
	g.Packages["click"].Requested = true

	// Autoremove check: click should survive, werkzeug should be orphan
	orphans := g.Orphans()
	for _, o := range orphans {
		if o == "click" {
			t.Error("click was marked requested, should not be in orphans")
		}
	}
	if len(orphans) != 1 || orphans[0] != "werkzeug" {
		t.Errorf("expected [werkzeug], got %v", orphans)
	}
}
