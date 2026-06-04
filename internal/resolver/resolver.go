package resolver

import (
	"context"
	"fmt"
	"sort"

	"github.com/kartikeyyadav/fpm/internal/client"
	"github.com/kartikeyyadav/fpm/internal/config"
	"github.com/kartikeyyadav/fpm/internal/env"
	"github.com/kartikeyyadav/fpm/internal/pep440"
	"github.com/kartikeyyadav/fpm/internal/pep508"
	"github.com/kartikeyyadav/fpm/internal/wheel"
	"github.com/kartikeyyadav/fpm/pkg/types"
)

type Resolver struct {
	client     *client.RegistryClient
	scanner    *env.ScanResult
	immutables []config.ImmutablePackage
	strategy   ResolutionStrategy
	ctx        context.Context
}

type ResolutionStrategy int

const (
	StrategyHighest  ResolutionStrategy = iota
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
	Client     *client.RegistryClient
	Scanner    *env.ScanResult
	Immutables []config.ImmutablePackage
	Strategy   ResolutionStrategy
}

func New(opts ResolverOptions) *Resolver {
	return &Resolver{
		client:     opts.Client,
		scanner:    opts.Scanner,
		immutables: opts.Immutables,
		strategy:   opts.Strategy,
		ctx:        context.Background(),
	}
}

func (r *Resolver) Resolve(requirements []pep508.Requirement) (*Resolution, error) {
	resolved := make(map[string]*ResolvedPackage)
	queue := make([]pep508.Requirement, len(requirements))
	copy(queue, requirements)

	visited := make(map[string]bool)

	for len(queue) > 0 {
		req := queue[0]
		queue = queue[1:]

		pkgName := req.Name.Normalized()
		if visited[pkgName] {
			continue
		}
		visited[pkgName] = true

		// Check immutable constraints
		if err := r.checkImmutable(req.Name); err != nil {
			immVer := r.getImmutableVersion(req.Name)
			if immVer != nil {
				if !req.Specifiers.Contains(*immVer) {
					return nil, &ImmutableConflictError{
						Package:   req.Name,
						Requested: req.Specifiers.String(),
						Pinned:    *immVer,
					}
				}
				// Use the immutable version
				resolved[pkgName] = &ResolvedPackage{
					Name:    req.Name,
					Version: *immVer,
				}
				continue
			}
		}

		// Resolve the package version
		pkg, err := r.resolvePackage(req)
		if err != nil {
			return nil, fmt.Errorf("failed to resolve %s: %w", req.Name.Raw(), err)
		}

		resolved[pkgName] = pkg

		// Add transitive dependencies to queue
		for _, dep := range pkg.Deps {
			depName := dep.Name.Normalized()
			if !visited[depName] {
				queue = append(queue, dep)
			}
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

func (r *Resolver) resolvePackage(req pep508.Requirement) (*ResolvedPackage, error) {
	detail, err := r.client.FetchPackageVersions(r.ctx, req.Name)
	if err != nil {
		return nil, err
	}

	// Collect available versions from wheel filenames
	type candidate struct {
		version pep440.Version
		file    client.SimpleFile
	}

	var candidates []candidate
	for _, f := range detail.Files {
		if f.IsYanked() {
			continue
		}
		whl, err := wheel.ParseFilename(f.Filename)
		if err != nil {
			continue // skip non-wheel or unparseable
		}
		if req.Specifiers.Contains(whl.Version) {
			candidates = append(candidates, candidate{
				version: whl.Version,
				file:    f,
			})
		}
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

	// Get dependency metadata
	var deps []pep508.Requirement
	metadata, err := r.fetchMetadata(best.file)
	if err == nil && metadata != nil {
		for _, reqStr := range metadata.RequiresDist {
			parsed, err := pep508.ParseRequirement(reqStr)
			if err == nil {
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
	// In a full implementation, this would download and parse METADATA
	// For now, return nil to indicate metadata not available
	// This will be enhanced in Phase 5 with actual wheel metadata fetching
	return nil, nil
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
