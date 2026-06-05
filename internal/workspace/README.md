# workspace

Workspace and monorepo support with pyproject.toml parsing.

## Key Types

- `Workspace` — workspace root with member discovery
- `Member` — workspace member (name, path)
- `PyProjectToml` — parsed pyproject.toml structure

## Features

- Discover workspace root by walking up directory tree
- Member discovery via glob patterns
- pyproject.toml reading/writing (PEP 621 `[project]` table)
- Add/remove dependencies programmatically

## Files

- `workspace.go` — discovery, member resolution, pyproject.toml management
