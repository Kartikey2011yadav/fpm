# errors

Structured error types with user-facing hints for actionable error messages.

## Key Types

- `FpmError` — error with message, cause chain, hint, and exit code

## Error Format

Every error includes what went wrong, why, and how to fix it:

```
error: Cannot install numpy 2.0.0
  cause: numpy is pinned as immutable at version 1.24.0 in fpm.toml
  hint: Remove the [immutable] entry for numpy to allow version changes
```

## Files

- `errors.go` — `FpmError`, `New`, `Wrap`, `WithHint`
