package platform

import (
	"runtime"
	"testing"
)

func TestCurrentPlatform(t *testing.T) {
	p := Current()
	if runtime.GOOS == "darwin" && p.OS != Darwin {
		t.Errorf("OS = %v, want Darwin", p.OS)
	}
	if runtime.GOOS == "linux" && p.OS != Linux {
		t.Errorf("OS = %v, want Linux", p.OS)
	}
}

func TestPlatformTag(t *testing.T) {
	p := Platform{OS: Linux, Arch: X86_64}
	tag := p.PythonPlatformTag()
	if tag != "manylinux_2_17_x86_64" {
		t.Errorf("tag = %q, want manylinux_2_17_x86_64", tag)
	}

	p2 := Platform{OS: Darwin, Arch: Aarch64}
	tag2 := p2.PythonPlatformTag()
	if tag2 != "macosx_11_0_arm64" {
		t.Errorf("tag = %q, want macosx_11_0_arm64", tag2)
	}

	p3 := Platform{OS: Windows, Arch: X86_64}
	tag3 := p3.PythonPlatformTag()
	if tag3 != "win_amd64" {
		t.Errorf("tag = %q, want win_amd64", tag3)
	}
}

func TestGenerateTags(t *testing.T) {
	tags := GenerateTags("311", Platform{OS: Linux, Arch: X86_64})
	if tags == nil {
		t.Fatal("GenerateTags returned nil")
	}
	if len(tags.Tags) == 0 {
		t.Error("no tags generated")
	}

	// First tag should be the most specific
	first := tags.Tags[0]
	if first.Python != "cp311" {
		t.Errorf("first python tag = %q, want cp311", first.Python)
	}
}

func TestWheelCompatibility(t *testing.T) {
	envTags := GenerateTags("311", Platform{OS: Linux, Arch: X86_64})

	// Pure wheel should be compatible
	pureTags := ParseWheelTags("py3", "none", "any")
	compatible, _ := envTags.Compatible(pureTags)
	if !compatible {
		t.Error("pure wheel should be compatible")
	}

	// Platform-specific for different arch should not
	wrongTags := ParseWheelTags("cp311", "cp311", "win_amd64")
	compatible2, _ := envTags.Compatible(wrongTags)
	if compatible2 {
		t.Error("windows wheel should not be compatible with linux")
	}
}

func TestSysPlatform(t *testing.T) {
	p := Platform{OS: Linux, Arch: X86_64}
	if p.SysPlatform() != "linux" {
		t.Errorf("SysPlatform = %q", p.SysPlatform())
	}
	if p.OSName() != "posix" {
		t.Errorf("OSName = %q", p.OSName())
	}
}
