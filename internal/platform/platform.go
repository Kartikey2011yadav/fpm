package platform

import (
	"fmt"
	"runtime"
	"strings"
)

type OS int

const (
	Linux OS = iota
	Darwin
	Windows
)

func (o OS) String() string {
	switch o {
	case Linux:
		return "linux"
	case Darwin:
		return "darwin"
	case Windows:
		return "windows"
	default:
		return "unknown"
	}
}

type Arch int

const (
	X86_64 Arch = iota
	Aarch64
	X86
	Armv7l
	Ppc64le
	S390x
)

func (a Arch) String() string {
	switch a {
	case X86_64:
		return "x86_64"
	case Aarch64:
		return "aarch64"
	case X86:
		return "i686"
	case Armv7l:
		return "armv7l"
	case Ppc64le:
		return "ppc64le"
	case S390x:
		return "s390x"
	default:
		return "unknown"
	}
}

type Platform struct {
	OS   OS
	Arch Arch
}

func Current() Platform {
	return Platform{
		OS:   currentOS(),
		Arch: currentArch(),
	}
}

func currentOS() OS {
	switch runtime.GOOS {
	case "linux":
		return Linux
	case "darwin":
		return Darwin
	case "windows":
		return Windows
	default:
		return Linux
	}
}

func currentArch() Arch {
	switch runtime.GOARCH {
	case "amd64":
		return X86_64
	case "arm64":
		return Aarch64
	case "386":
		return X86
	case "arm":
		return Armv7l
	case "ppc64le":
		return Ppc64le
	case "s390x":
		return S390x
	default:
		return X86_64
	}
}

func (p Platform) PythonPlatformTag() string {
	switch p.OS {
	case Linux:
		return p.linuxPlatformTag()
	case Darwin:
		return p.darwinPlatformTag()
	case Windows:
		return p.windowsPlatformTag()
	default:
		return "any"
	}
}

func (p Platform) linuxPlatformTag() string {
	arch := p.Arch.String()
	return fmt.Sprintf("manylinux_2_17_%s", arch)
}

func (p Platform) darwinPlatformTag() string {
	arch := p.Arch.String()
	if p.Arch == Aarch64 {
		arch = "arm64"
	}
	return fmt.Sprintf("macosx_11_0_%s", arch)
}

func (p Platform) windowsPlatformTag() string {
	switch p.Arch {
	case X86_64:
		return "win_amd64"
	case X86:
		return "win32"
	case Aarch64:
		return "win_arm64"
	default:
		return "win_amd64"
	}
}

func (p Platform) SysPlatform() string {
	switch p.OS {
	case Linux:
		return "linux"
	case Darwin:
		return "darwin"
	case Windows:
		return "win32"
	default:
		return "unknown"
	}
}

func (p Platform) OSName() string {
	switch p.OS {
	case Windows:
		return "nt"
	default:
		return "posix"
	}
}

func (p Platform) PlatformSystem() string {
	switch p.OS {
	case Linux:
		return "Linux"
	case Darwin:
		return "Darwin"
	case Windows:
		return "Windows"
	default:
		return ""
	}
}

func (p Platform) PlatformMachine() string {
	switch p.Arch {
	case X86_64:
		return "x86_64"
	case Aarch64:
		if p.OS == Darwin {
			return "arm64"
		}
		return "aarch64"
	case X86:
		return "i686"
	default:
		return p.Arch.String()
	}
}

// ManylinuxVersions returns manylinux compatibility versions in descending order.
func ManylinuxVersions() []string {
	return []string{
		"manylinux_2_35",
		"manylinux_2_34",
		"manylinux_2_31",
		"manylinux_2_28",
		"manylinux_2_24",
		"manylinux_2_17",
		"manylinux2014",
		"manylinux2010",
		"manylinux1",
	}
}

// MacOSVersions returns macOS compatibility versions in descending order.
func MacOSVersions(major, minor int) []string {
	var versions []string
	for m := minor; m >= 0; m-- {
		versions = append(versions, fmt.Sprintf("macosx_%d_%d", major, m))
	}
	if major >= 11 {
		for maj := major - 1; maj >= 10; maj-- {
			if maj == 10 {
				for m := 16; m >= 9; m-- {
					versions = append(versions, fmt.Sprintf("macosx_%d_%d", maj, m))
				}
			}
		}
	}
	return versions
}

// NormalizePlatformTag normalizes platform tag variants.
func NormalizePlatformTag(tag string) string {
	tag = strings.ReplaceAll(tag, "-", "_")
	tag = strings.ReplaceAll(tag, ".", "_")
	return strings.ToLower(tag)
}
