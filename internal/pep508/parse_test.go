package pep508

import (
	"testing"
)

func TestParseSimple(t *testing.T) {
	tests := []struct {
		input string
		name  string
	}{
		{"requests", "requests"},
		{"numpy", "numpy"},
		{"my-package", "my-package"},
		{"my_package", "my-package"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			req, err := ParseRequirement(tt.input)
			if err != nil {
				t.Fatalf("ParseRequirement(%q) error: %v", tt.input, err)
			}
			if req.Name.Normalized() != tt.name {
				t.Errorf("name = %q, want %q", req.Name.Normalized(), tt.name)
			}
		})
	}
}

func TestParseWithVersion(t *testing.T) {
	tests := []struct {
		input string
		name  string
		specs string
	}{
		{"requests>=2.28.0", "requests", ">=2.28.0"},
		{"numpy ==1.24.0", "numpy", "==1.24.0"},
		{"flask>=2.0,<3.0", "flask", ">=2.0, <3.0"},
		{"django~=4.2", "django", "~=4.2"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			req, err := ParseRequirement(tt.input)
			if err != nil {
				t.Fatalf("ParseRequirement(%q) error: %v", tt.input, err)
			}
			if req.Name.Normalized() != tt.name {
				t.Errorf("name = %q, want %q", req.Name.Normalized(), tt.name)
			}
			if req.Specifiers.String() != tt.specs {
				t.Errorf("specifiers = %q, want %q", req.Specifiers.String(), tt.specs)
			}
		})
	}
}

func TestParseWithExtras(t *testing.T) {
	tests := []struct {
		input  string
		name   string
		extras []string
	}{
		{"requests[security]", "requests", []string{"security"}},
		{"package[extra1,extra2]", "package", []string{"extra1", "extra2"}},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			req, err := ParseRequirement(tt.input)
			if err != nil {
				t.Fatalf("ParseRequirement(%q) error: %v", tt.input, err)
			}
			if req.Name.Normalized() != tt.name {
				t.Errorf("name = %q, want %q", req.Name.Normalized(), tt.name)
			}
			if len(req.Extras) != len(tt.extras) {
				t.Fatalf("extras count = %d, want %d", len(req.Extras), len(tt.extras))
			}
			for i, e := range tt.extras {
				if req.Extras[i].Normalized() != e {
					t.Errorf("extras[%d] = %q, want %q", i, req.Extras[i].Normalized(), e)
				}
			}
		})
	}
}

func TestParseWithMarkers(t *testing.T) {
	tests := []struct {
		input     string
		name      string
		hasMarker bool
	}{
		{`requests>=2.28; python_version >= "3.8"`, "requests", true},
		{`numpy; sys_platform == "linux"`, "numpy", true},
		{"flask>=2.0", "flask", false},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			req, err := ParseRequirement(tt.input)
			if err != nil {
				t.Fatalf("ParseRequirement(%q) error: %v", tt.input, err)
			}
			if req.Name.Normalized() != tt.name {
				t.Errorf("name = %q, want %q", req.Name.Normalized(), tt.name)
			}
			if (req.Marker != nil) != tt.hasMarker {
				t.Errorf("hasMarker = %v, want %v", req.Marker != nil, tt.hasMarker)
			}
		})
	}
}

func TestMarkerEvaluation(t *testing.T) {
	env := MarkerEnvironment{
		OSName:             "posix",
		SysPlatform:        "linux",
		PlatformSystem:     "Linux",
		PythonVersion:      "3.11",
		PythonFullVersion:  "3.11.5",
		ImplementationName: "cpython",
	}

	tests := []struct {
		input string
		match bool
	}{
		{`requests; python_version >= "3.8"`, true},
		{`requests; python_version < "3.8"`, false},
		{`requests; sys_platform == "linux"`, true},
		{`requests; sys_platform == "win32"`, false},
		{`requests; python_version >= "3.8" and sys_platform == "linux"`, true},
		{`requests; python_version >= "3.8" and sys_platform == "win32"`, false},
		{`requests; python_version < "3.8" or sys_platform == "linux"`, true},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			req, err := ParseRequirement(tt.input)
			if err != nil {
				t.Fatalf("ParseRequirement(%q) error: %v", tt.input, err)
			}
			if got := req.EvaluateMarkers(env); got != tt.match {
				t.Errorf("EvaluateMarkers() = %v, want %v", got, tt.match)
			}
		})
	}
}

func TestParseURL(t *testing.T) {
	input := "package @ https://example.com/package-1.0.tar.gz"
	req, err := ParseRequirement(input)
	if err != nil {
		t.Fatalf("ParseRequirement(%q) error: %v", input, err)
	}
	if req.Name.Normalized() != "package" {
		t.Errorf("name = %q, want %q", req.Name.Normalized(), "package")
	}
	if req.URL != "https://example.com/package-1.0.tar.gz" {
		t.Errorf("URL = %q, want %q", req.URL, "https://example.com/package-1.0.tar.gz")
	}
}
