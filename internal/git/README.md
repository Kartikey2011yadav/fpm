# git

Git dependency support for packages hosted in git repositories.

## Key Types

- `GitSource` — git URL + reference + subdirectory
- `CloneResult` — cloned repo path + resolved commit SHA

## Supported URL Formats

- `git+https://github.com/user/repo.git@v1.0`
- `git+ssh://git@github.com/user/repo.git@main`
- `git+https://...#subdirectory=subdir`

## Features

- Clone with shallow depth for speed
- Cache cloned repos to avoid re-cloning
- Resolve branches, tags, and commit references
- Support subdirectory specification

## Files

- `git.go` — clone, fetch, checkout, URL parsing, caching
