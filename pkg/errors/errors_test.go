package errors

import (
	"fmt"
	"strings"
	"testing"
)

func TestFpmError(t *testing.T) {
	err := New("something failed")
	if !strings.Contains(err.Error(), "something failed") {
		t.Errorf("error = %q", err.Error())
	}
}

func TestFpmErrorWithCause(t *testing.T) {
	cause := fmt.Errorf("network timeout")
	err := Wrap(cause, "download failed")

	if !strings.Contains(err.Error(), "download failed") {
		t.Error("missing message")
	}
	if !strings.Contains(err.Error(), "network timeout") {
		t.Error("missing cause")
	}
}

func TestFpmErrorWithHint(t *testing.T) {
	err := New("cannot install numpy 2.0.0")
	err = WithHint(err, "Remove the [immutable] entry for numpy")

	output := err.Error()
	if !strings.Contains(output, "hint:") {
		t.Error("missing hint prefix")
	}
	if !strings.Contains(output, "Remove the [immutable]") {
		t.Error("missing hint content")
	}
}

func TestNewf(t *testing.T) {
	err := Newf("package %q version %s", "numpy", "1.24.0")
	if !strings.Contains(err.Error(), `"numpy"`) {
		t.Errorf("error = %q", err.Error())
	}
}

func TestUnwrap(t *testing.T) {
	cause := fmt.Errorf("root cause")
	err := Wrap(cause, "wrapper")

	unwrapped := err.Unwrap()
	if unwrapped != cause {
		t.Error("Unwrap didn't return cause")
	}
}
