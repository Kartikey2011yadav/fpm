package resolver

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"github.com/kartikeyyadav/fpm/internal/client"
	"github.com/kartikeyyadav/fpm/internal/config"
	"github.com/kartikeyyadav/fpm/internal/env"
	"github.com/kartikeyyadav/fpm/internal/pep440"
	"github.com/kartikeyyadav/fpm/internal/pep508"
	"github.com/kartikeyyadav/fpm/internal/platform"
	"github.com/kartikeyyadav/fpm/internal/wheel"
	"github.com/kartikeyyadav/fpm/pkg/types"
)

type Resolver struct {
	client     *client.RegistryClient
	scanner    *env.ScanResult
	immutables []config.ImmutablePackage
	strategy   ResolutionStrategy
	tagSet     *platform.TagSet
	ctx        context.Context
}

type ResolutionStrategy int

const (
	StrategyHighest ResolutionStrategy = iota
	StrategyLowest
	StrategyInstalled
)

type ResolvedPackage struct {
	Name    types.PackageName
	Version pep440.Version
	URL     string
	Hash    string
	Deps    []pep508.Requirement
}

type Resolution struct {
	Packages []ResolvedPackage
}

type ResolverOptions struct {
	Client        *client.RegistryClient
	Scanner       *env.ScanResult
	Immutables    []config.ImmutablePackage
	Strategy      ResolutionStrategy
	PythonVersion string
}

func New(opts ResolverOptions) *Resolver {
	pyVer := opts.PythonVersion
	if pyVer == "" {
		pyVer = "311"
	}
	tagSet := platform.GenerateTags(pyVer, platform.Current())

	return &Resolver{
		client:     opts.Client,
		scanner:    opts.Scanner,
		immutables: opts.Immutables,
		strategy:   opts.Strategy,
		tagSet:     tagSet,
		ctx:        context.Background(),
	}
}

func (r *Resolver) Resolve(requirements []pep508.Requirement) (*Resolution, error) {
	resolved := make(map[string]*ResolvedPackage)
	queue := make([]pep508.Requirement, len(requirements))
	copy(queue, requirements)

	// Track all constraints per package for conflict detection
	constraints := make(map[string][]pep508.Requirement)

	for len(queue) > 0 {
		req := queue[0]
		queue = queue[1:]

		pkgName := req.Name.Normalized()

		// Record constraint
		constraints[pkgName] = append(constraints[pkgName], req)

		// If already resolved, verify the resolved version satisfies this new constraint
		if existing, ok := resolved[pkgName]; ok {
			if len(req.Specifiers) > 0 && !req.Specifiers.Contains(existing.Version) {
				return nil, &ConflictError{
					Package:     req.Name,
					Resolved:    existing.Version,
					Constraints: constraints[pkgName],
				}
			}
			continue
		}

		// Merge all known specifiers for this package
		mergedReq := req
		for _, prev := range constraints[pkgName][:len(constraints[pkgName])-1] {
			mergedReq.Specifiers = append(mergedReq.Specifiers, prev.Specifiers...)
		}

		// Check immutable constraints
		if err := r.checkImmutable(req.Name); err != nil {
			immVer := r.getImmutableVersion(req.Name)
			if immVer != nil {
				if !mergedReq.Specifiers.Contains(*immVer) {
					return nil, &ImmutableConflictError{
						Package:   req.Name,
						Requested: mergedReq.Specifiers.String(),
						Pinned:    *immVer,
					}
				}
				resolved[pkgName] = &ResolvedPackage{
					Name:    req.Name,
					Version: *immVer,
				}
				continue
			}
		}

		// Resolve the package version using merged constraints
		pkg, err := r.resolvePackage(mergedReq)
		if err != nil {
			return nil, fmt.Errorf("failed to resolve %s: %w", req.Name.Raw(), err)
		}

		resolved[pkgName] = pkg

		// Add transitive dependencies to queue
		for _, dep := range pkg.Deps {
			queue = append(queue, dep)
		}
	}

	// Build result
	result := &Resolution{}
	for _, pkg := range resolved {
		result.Packages = append(result.Packages, *pkg)
	}

	// Sort for deterministic output
	sort.Slice(result.Packages, func(i, j int) bool {
		return result.Packages[i].Name.Normalized() < result.Packages[j].Name.Normalized()
	})

	return result, nil
}

// ConflictError indicates incompatible version requirements for a package.
type ConflictError struct {
	Package     types.PackageName
	Resolved    pep440.Version
	Constraints []pep508.Requirement
}

func (e *ConflictError) Error() string {
	var parts []string
	for _, c := range e.Constraints {
		if len(c.Specifiers) > 0 {
			parts = append(parts, fmt.Sprintf("%s (requires %s%s)", c.Name.Raw(), c.Name.Raw(), c.Specifiers.String()))
		}
	}
	return fmt.Sprintf(
		"version conflict for %s: resolved %s but incompatible constraints exist: %s",
		e.Package.Raw(), e.Resolved.String(), fmt.Sprintf("%v", parts),
	)
}

func (r *Resolver) resolvePackage(req pep508.Requirement) (*ResolvedPackage, error) {
	detail, err := r.client.FetchPackageVersions(r.ctx, req.Name)
	if err != nil {
		return nil, err
	}

	// Collect available versions from wheel filenames
	type candidate struct {
		version  pep440.Version
		file     client.SimpleFile
		priority int
		pure     bool
	}

	var candidates []candidate
	for _, f := range detail.Files {
		if f.IsYanked() {
			continue
		}
		whl, err := wheel.ParseFilename(f.Filename)
		if err != nil {
			continue
		}
		if !req.Specifiers.Contains(whl.Version) {
			continue
		}
		// Check platform compatibility
		wheelTags := whl.Tags()
		compatible, priority := r.tagSet.Compatible(wheelTags)
		if !compatible {
			continue
		}
		candidates = append(candidates, candidate{
			version:  whl.Version,
			file:     f,
			priority: priority,
			pure:     whl.IsPureWheel(),
		})
	}
	// Deduplicate: keep best candidate per version (prefer pure, then best priority)
	versionBest := make(map[string]candidate)
	for _, c := range candidates {
		key := c.version.String()
		existing, exists := versionBest[key]
		if !exists {
			versionBest[key] = c
		} else if c.pure && !existing.pure {
			versionBest[key] = c
		} else if c.pure == existing.pure && c.priority < existing.priority {
			versionBest[key] = c
		}
	}
	candidates = candidates[:0]
	for _, c := range versionBest {
		candidates = append(candidates, c)
	}

	if len(candidates) == 0 {
		return nil, fmt.Errorf("no compatible version found for %s %s",
			req.Name.Raw(), req.Specifiers.String())
	}

	// Sort by version
	sort.Slice(candidates, func(i, j int) bool {
		cmp := pep440.Compare(candidates[i].version, candidates[j].version)
		if r.strategy == StrategyLowest {
			return cmp < 0
		}
		return cmp > 0
	})

	// Pick the best candidate
	best := candidates[0]

	// Get dependency metadata from the selected wheel
	var deps []pep508.Requirement
	metadata, err := r.fetchMetadata(best.file)
	if err == nil && metadata != nil {
		for _, reqStr := range metadata.RequiresDist {
			parsed, parseErr := pep508.ParseRequirement(reqStr)
			if parseErr == nil {
				// Skip extras-only dependencies (e.g. ; extra == "security")
				if parsed.Marker != nil {
					// Evaluate with empty extra — skip if marker requires an extra
					env := pep508.MarkerEnvironment{}
					if !parsed.EvaluateMarkers(env) {
						continue
					}
				}
				deps = append(deps, parsed)
			}
		}
	}

	return &ResolvedPackage{
		Name:    req.Name,
		Version: best.version,
		URL:     best.file.URL,
		Hash:    best.file.SHA256(),
		Deps:    deps,
	}, nil
}

func (r *Resolver) fetchMetadata(file client.SimpleFile) (*wheel.Metadata, error) {
	if file.URL == "" {
		return nil, fmt.Errorf("no URL for metadata fetch")
	}

	// Download the wheel to a temp location and extract metadata
	tmpDir, err := os.MkdirTemp("", "fpm-metadata-*")
	if err != nil {
		return nil, err
	}
	defer os.RemoveAll(tmpDir)

	wheelPath := filepath.Join(tmpDir, file.Filename)
	if err := r.client.DownloadWheel(r.ctx, file, wheelPath); err != nil {
		return nil, fmt.Errorf("failed to download for metadata: %w", err)
	}

	return wheel.ReadMetadataFromZip(wheelPath)
}

func (r *Resolver) checkImmutable(name types.PackageName) error {
	for _, im := range r.immutables {
		if types.NewPackageName(im.Name).Equal(name) {
			return fmt.Errorf("immutable")
		}
	}
	return nil
}

func (r *Resolver) getImmutableVersion(name types.PackageName) *pep440.Version {
	for _, im := range r.immutables {
		if types.NewPackageName(im.Name).Equal(name) {
			ver, err := pep440.Parse(im.Version)
			if err != nil {
				return nil
			}
			return &ver
		}
	}
	return nil
}

// Errors

type ImmutableConflictError struct {
	Package   types.PackageName
	Requested string
	Pinned    pep440.Version
}

func (e *ImmutableConflictError) Error() string {
	return fmt.Sprintf(
		"cannot install %s %s: pinned as immutable at version %s in fpm.toml",
		e.Package.Raw(), e.Requested, e.Pinned.String(),
	)
}

type ResolutionError struct {
	Package types.PackageName
	Message string
}

func (e *ResolutionError) Error() string {
	return fmt.Sprintf("resolution failed for %s: %s", e.Package.Raw(), e.Message)
}
