package wheel

import (
	"testing"
)

func TestParseFilename(t *testing.T) {
	tests := []struct {
		input   string
		name    string
		version string
		python  []string
		abi     []string
		plat    []string
		isPure  bool
	}{
		{
			"numpy-1.24.0-cp311-cp311-manylinux_2_17_x86_64.manylinux2014_x86_64.whl",
			"numpy", "1.24.0",
			[]string{"cp311"}, []string{"cp311"},
			[]string{"manylinux_2_17_x86_64", "manylinux2014_x86_64"},
			false,
		},
		{
			"requests-2.31.0-py3-none-any.whl",
			"requests", "2.31.0",
			[]string{"py3"}, []string{"none"}, []string{"any"},
			true,
		},
		{
			"torch-2.1.0-cp311-cp311-macosx_11_0_arm64.whl",
			"torch", "2.1.0",
			[]string{"cp311"}, []string{"cp311"}, []string{"macosx_11_0_arm64"},
			false,
		},
		{
			"pip-23.3.1-py3-none-any.whl",
			"pip", "23.3.1",
			[]string{"py3"}, []string{"none"}, []string{"any"},
			true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			w, err := ParseFilename(tt.input)
			if err != nil {
				t.Fatalf("ParseFilename(%q) error: %v", tt.input, err)
			}
			if w.Name.Normalized() != tt.name {
				t.Errorf("name = %q, want %q", w.Name.Normalized(), tt.name)
			}
			if w.Version.String() != tt.version {
				t.Errorf("version = %q, want %q", w.Version.String(), tt.version)
			}
			if !sliceEqual(w.PythonTag, tt.python) {
				t.Errorf("python = %v, want %v", w.PythonTag, tt.python)
			}
			if !sliceEqual(w.AbiTag, tt.abi) {
				t.Errorf("abi = %v, want %v", w.AbiTag, tt.abi)
			}
			if !sliceEqual(w.Platform, tt.plat) {
				t.Errorf("platform = %v, want %v", w.Platform, tt.plat)
			}
			if w.IsPureWheel() != tt.isPure {
				t.Errorf("isPure = %v, want %v", w.IsPureWheel(), tt.isPure)
			}
		})
	}
}

func TestParseFilenameInvalid(t *testing.T) {
	invalids := []string{
		"not-a-wheel.tar.gz",
		"too-few-parts.whl",
		"a-b-c-d-e-f-g-h.whl",
	}

	for _, input := range invalids {
		_, err := ParseFilename(input)
		if err == nil {
			t.Errorf("ParseFilename(%q) expected error, got nil", input)
		}
	}
}

func sliceEqual(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
