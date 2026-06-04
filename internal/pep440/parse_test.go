package pep440

import (
	"testing"
)

func TestParseBasic(t *testing.T) {
	tests := []struct {
		input   string
		want    string
		epoch   int
		release []int
	}{
		{"1.0", "1.0", 0, []int{1, 0}},
		{"1.0.0", "1.0.0", 0, []int{1, 0, 0}},
		{"1.2.3.4", "1.2.3.4", 0, []int{1, 2, 3, 4}},
		{"2!1.0", "2!1.0", 2, []int{1, 0}},
		{"0.9", "0.9", 0, []int{0, 9}},
		{"22.3", "22.3", 0, []int{22, 3}},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			v, err := Parse(tt.input)
			if err != nil {
				t.Fatalf("Parse(%q) error: %v", tt.input, err)
			}
			if v.String() != tt.want {
				t.Errorf("Parse(%q).String() = %q, want %q", tt.input, v.String(), tt.want)
			}
			if v.Epoch != tt.epoch {
				t.Errorf("Parse(%q).Epoch = %d, want %d", tt.input, v.Epoch, tt.epoch)
			}
			if len(v.Release) != len(tt.release) {
				t.Fatalf("Parse(%q).Release len = %d, want %d", tt.input, len(v.Release), len(tt.release))
			}
			for i, seg := range tt.release {
				if v.Release[i] != seg {
					t.Errorf("Parse(%q).Release[%d] = %d, want %d", tt.input, i, v.Release[i], seg)
				}
			}
		})
	}
}

func TestParsePreRelease(t *testing.T) {
	tests := []struct {
		input string
		want  string
		kind  PreReleaseKind
		num   int
	}{
		{"1.0a1", "1.0a1", PreAlpha, 1},
		{"1.0alpha2", "1.0a2", PreAlpha, 2},
		{"1.0b3", "1.0b3", PreBeta, 3},
		{"1.0beta4", "1.0b4", PreBeta, 4},
		{"1.0rc1", "1.0rc1", PreRC, 1},
		{"1.0c2", "1.0rc2", PreRC, 2},
		{"1.0preview3", "1.0rc3", PreRC, 3},
		{"1.0.a1", "1.0a1", PreAlpha, 1},
		{"1.0-b2", "1.0b2", PreBeta, 2},
		{"1.0_rc3", "1.0rc3", PreRC, 3},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			v, err := Parse(tt.input)
			if err != nil {
				t.Fatalf("Parse(%q) error: %v", tt.input, err)
			}
			if v.String() != tt.want {
				t.Errorf("Parse(%q).String() = %q, want %q", tt.input, v.String(), tt.want)
			}
			if v.Pre == nil {
				t.Fatalf("Parse(%q).Pre is nil", tt.input)
			}
			if v.Pre.Kind != tt.kind {
				t.Errorf("Parse(%q).Pre.Kind = %v, want %v", tt.input, v.Pre.Kind, tt.kind)
			}
			if v.Pre.Number != tt.num {
				t.Errorf("Parse(%q).Pre.Number = %d, want %d", tt.input, v.Pre.Number, tt.num)
			}
		})
	}
}

func TestParsePostDev(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"1.0.post1", "1.0.post1"},
		{"1.0.dev5", "1.0.dev5"},
		{"1.0a1.post2", "1.0a1.post2"},
		{"1.0.post1.dev2", "1.0.post1.dev2"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			v, err := Parse(tt.input)
			if err != nil {
				t.Fatalf("Parse(%q) error: %v", tt.input, err)
			}
			if v.String() != tt.want {
				t.Errorf("Parse(%q).String() = %q, want %q", tt.input, v.String(), tt.want)
			}
		})
	}
}

func TestParseLocal(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"1.0+local", "1.0+local"},
		{"1.0+abc.1", "1.0+abc.1"},
		{"1.0+ubuntu1", "1.0+ubuntu1"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			v, err := Parse(tt.input)
			if err != nil {
				t.Fatalf("Parse(%q) error: %v", tt.input, err)
			}
			if v.String() != tt.want {
				t.Errorf("Parse(%q).String() = %q, want %q", tt.input, v.String(), tt.want)
			}
		})
	}
}

func TestCompare(t *testing.T) {
	ordered := []string{
		"1.0.dev1",
		"1.0a1",
		"1.0a2",
		"1.0b1",
		"1.0rc1",
		"1.0",
		"1.0.post1",
		"1.1.dev1",
		"1.1",
		"2.0",
	}

	for i := 0; i < len(ordered)-1; i++ {
		a, _ := Parse(ordered[i])
		b, _ := Parse(ordered[i+1])
		if !a.LessThan(b) {
			t.Errorf("expected %s < %s", ordered[i], ordered[i+1])
		}
	}
}

func TestSpecifiers(t *testing.T) {
	tests := []struct {
		spec    string
		version string
		match   bool
	}{
		{">=1.0", "1.0", true},
		{">=1.0", "0.9", false},
		{">=1.0", "2.0", true},
		{"<2.0", "1.9", true},
		{"<2.0", "2.0", false},
		{"==1.0.*", "1.0.1", true},
		{"==1.0.*", "1.1.0", false},
		{"~=1.4", "1.5", true},
		{"~=1.4", "2.0", false},
		{"~=1.4.2", "1.4.5", true},
		{"~=1.4.2", "1.5.0", false},
		{"!=1.5", "1.4", true},
		{"!=1.5", "1.5", false},
	}

	for _, tt := range tests {
		t.Run(tt.spec+"_"+tt.version, func(t *testing.T) {
			s, err := ParseSpecifier(tt.spec)
			if err != nil {
				t.Fatalf("ParseSpecifier(%q) error: %v", tt.spec, err)
			}
			v, err := Parse(tt.version)
			if err != nil {
				t.Fatalf("Parse(%q) error: %v", tt.version, err)
			}
			if got := s.Contains(v); got != tt.match {
				t.Errorf("Specifier(%q).Contains(%q) = %v, want %v", tt.spec, tt.version, got, tt.match)
			}
		})
	}
}

func TestMultipleSpecifiers(t *testing.T) {
	specs, err := ParseSpecifiers(">=1.0, <2.0, !=1.5")
	if err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		version string
		match   bool
	}{
		{"0.9", false},
		{"1.0", true},
		{"1.4", true},
		{"1.5", false},
		{"1.6", true},
		{"2.0", false},
	}

	for _, tt := range tests {
		v, _ := Parse(tt.version)
		if got := specs.Contains(v); got != tt.match {
			t.Errorf("specs.Contains(%q) = %v, want %v", tt.version, got, tt.match)
		}
	}
}
