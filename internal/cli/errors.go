package cli

import (
	"errors"
	"fmt"
	"os"
	"strings"

	fpmErrors "github.com/kartikeyyadav/fpm/pkg/errors"
)

func formatError(err error) {
	if err == nil {
		return
	}

	useColor := shouldColorErrors()

	var fpmErr *fpmErrors.FpmError
	if errors.As(err, &fpmErr) {
		printStyledError(fpmErr, useColor)
		return
	}

	// Plain error — just print with red prefix
	if useColor {
		fmt.Fprintf(os.Stderr, "\n  \033[1;31m✗ error:\033[0m %s\n\n", err.Error())
	} else {
		fmt.Fprintf(os.Stderr, "\n  error: %s\n\n", err.Error())
	}
}

func printStyledError(e *fpmErrors.FpmError, useColor bool) {
	if useColor {
		fmt.Fprintf(os.Stderr, "\n  \033[1;31m✗ error:\033[0m \033[1m%s\033[0m\n", e.Message)
	} else {
		fmt.Fprintf(os.Stderr, "\n  error: %s\n", e.Message)
	}

	if e.Cause != nil {
		causeMsg := e.Cause.Error()
		if useColor {
			fmt.Fprintf(os.Stderr, "    \033[2mCaused by: %s\033[0m\n", causeMsg)
		} else {
			fmt.Fprintf(os.Stderr, "    Caused by: %s\n", causeMsg)
		}
	}

	if e.Hint != "" {
		lines := strings.Split(e.Hint, "\n")
		if useColor {
			fmt.Fprintf(os.Stderr, "\n    \033[36mhint:\033[0m %s\n", lines[0])
			for _, line := range lines[1:] {
				fmt.Fprintf(os.Stderr, "          %s\n", line)
			}
		} else {
			fmt.Fprintf(os.Stderr, "\n    hint: %s\n", lines[0])
			for _, line := range lines[1:] {
				fmt.Fprintf(os.Stderr, "          %s\n", line)
			}
		}
	}

	fmt.Fprintln(os.Stderr)
}

func shouldColorErrors() bool {
	if os.Getenv("NO_COLOR") != "" {
		return false
	}
	switch flagColor {
	case "never":
		return false
	case "always":
		return true
	default:
		fi, _ := os.Stderr.Stat()
		return fi != nil && fi.Mode()&os.ModeCharDevice != 0
	}
}
