# resolver

Dependency resolver using the PubGrub algorithm with immutable package
constraints.

## Key Types

- `Resolver` — main resolver orchestrator
- `Resolution` — resolved package set
- `ResolvedPackage` — package with chosen version, URL, hash, and dependencies
- `ImmutableConflictError` — error when immutable constraint is violated

## Resolution Strategies

- `StrategyHighest` — prefer highest compatible version (default)
- `StrategyLowest` — prefer lowest compatible version
- `StrategyInstalled` — prefer already-installed versions

## Immutable Constraints

Packages listed in `fpm.toml [immutable]` are injected as hard constraints. If a
transitive dependency requires a different version, the resolver backtracks or
fails with a clear error.

## Files

- `resolver.go` — resolution loop, version selection, immutable enforcement
