package errors

import (
	"fmt"
	"strings"
)

type ExitCode int

const (
	ExitSuccess    ExitCode = 0
	ExitFailure    ExitCode = 1
	ExitUsageError ExitCode = 2
)

type FpmError struct {
	Message  string
	Cause    error
	Hint     string
	ExitCode ExitCode
}

func (e *FpmError) Error() string {
	var b strings.Builder
	b.WriteString("error: ")
	b.WriteString(e.Message)
	if e.Cause != nil {
		b.WriteString("\n  cause: ")
		b.WriteString(e.Cause.Error())
	}
	if e.Hint != "" {
		b.WriteString("\n  hint: ")
		b.WriteString(e.Hint)
	}
	return b.String()
}

func (e *FpmError) Unwrap() error {
	return e.Cause
}

func New(message string) *FpmError {
	return &FpmError{
		Message:  message,
		ExitCode: ExitFailure,
	}
}

func Newf(format string, args ...interface{}) *FpmError {
	return &FpmError{
		Message:  fmt.Sprintf(format, args...),
		ExitCode: ExitFailure,
	}
}

func Wrap(err error, message string) *FpmError {
	return &FpmError{
		Message:  message,
		Cause:    err,
		ExitCode: ExitFailure,
	}
}

func WithHint(err *FpmError, hint string) *FpmError {
	err.Hint = hint
	return err
}
