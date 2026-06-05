# ui

Terminal output formatting with colors, progress bars, and multiple output
modes.

## Key Types

- `Output` — output controller (level, color mode, JSON mode, TTY detection)
- `ProgressBar` — progress indicator with percentage and bar
- `Spinner` — activity indicator for indeterminate operations

## Features

- Respects `NO_COLOR` environment variable
- Auto-detects TTY for color support
- Multiple verbosity levels (silent, quiet, default, verbose)
- JSON output mode for scripting
- Table formatting with column alignment
- Progress bars with Unicode block characters

## Files

- `output.go` — `Output`, `ProgressBar`, `Spinner`, `Table`
