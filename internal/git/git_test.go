package git

import (
	"testing"
)

func TestParseGitURL(t *testing.T) {
	tests := []struct {
		input  string
		url    string
		ref    string
		subdir string
	}{
		{
			"git+https://github.com/user/repo.git@v1.0",
			"https://github.com/user/repo.git",
			"v1.0",
			"",
		},
		{
			"git+https://github.com/user/repo.git@main",
			"https://github.com/user/repo.git",
			"main",
			"",
		},
		{
			"git+https://github.com/user/repo.git",
			"https://github.com/user/repo.git",
			"",
			"",
		},
		{
			"git+https://github.com/user/repo.git@v2.0#subdirectory=src",
			"https://github.com/user/repo.git",
			"v2.0",
			"src",
		},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			source, err := ParseGitURL(tt.input)
			if err != nil {
				t.Fatalf("ParseGitURL(%q) error: %v", tt.input, err)
			}
			if source.URL != tt.url {
				t.Errorf("URL = %q, want %q", source.URL, tt.url)
			}
			if source.Reference != tt.ref {
				t.Errorf("Reference = %q, want %q", source.Reference, tt.ref)
			}
			if source.Subdirectory != tt.subdir {
				t.Errorf("Subdirectory = %q, want %q", source.Subdirectory, tt.subdir)
			}
		})
	}
}

func TestIsGitURL(t *testing.T) {
	tests := []struct {
		input string
		want  bool
	}{
		{"git+https://github.com/user/repo.git", true},
		{"git+ssh://git@github.com/user/repo.git", true},
		{"git://github.com/user/repo.git", true},
		{"https://github.com/user/repo.git", true},
		{"https://pypi.org/simple/numpy/", false},
		{"numpy>=1.24", false},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := IsGitURL(tt.input)
			if got != tt.want {
				t.Errorf("IsGitURL(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}
