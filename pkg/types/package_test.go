package types

import (
	"testing"
)

func TestPackageNameNormalization(t *testing.T) {
	tests := []struct {
		input      string
		normalized string
	}{
		{"requests", "requests"},
		{"My-Package", "my-package"},
		{"my_package", "my-package"},
		{"My.Package", "my-package"},
		{"NUMPY", "numpy"},
		{"Flask-RESTful", "flask-restful"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			p := NewPackageName(tt.input)
			if p.Normalized() != tt.normalized {
				t.Errorf("Normalized(%q) = %q, want %q", tt.input, p.Normalized(), tt.normalized)
			}
		})
	}
}

func TestPackageNameEquality(t *testing.T) {
	a := NewPackageName("My-Package")
	b := NewPackageName("my_package")
	c := NewPackageName("other")

	if !a.Equal(b) {
		t.Error("My-Package should equal my_package")
	}
	if a.Equal(c) {
		t.Error("My-Package should not equal other")
	}
}

func TestPackageNameRaw(t *testing.T) {
	p := NewPackageName("Flask-RESTful")
	if p.Raw() != "Flask-RESTful" {
		t.Errorf("Raw() = %q, want 'Flask-RESTful'", p.Raw())
	}
}

func TestHashDigest(t *testing.T) {
	h := HashDigest{Algorithm: "sha256", Value: "abc123"}
	if h.String() != "sha256:abc123" {
		t.Errorf("String() = %q", h.String())
	}
}
