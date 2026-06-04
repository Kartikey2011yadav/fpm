package platform

import (
	"fmt"
	"sort"
)

type Tag struct {
	Python   string
	Abi      string
	Platform string
}

func (t Tag) String() string {
	return fmt.Sprintf("%s-%s-%s", t.Python, t.Abi, t.Platform)
}

type TagSet struct {
	Tags     []Tag
	priority map[string]int
}

func (ts *TagSet) Compatible(wheelTags []Tag) (bool, int) {
	best := -1
	for _, wt := range wheelTags {
		if p, ok := ts.priority[wt.String()]; ok {
			if best == -1 || p < best {
				best = p
			}
		}
	}
	return best >= 0, best
}

func GenerateTags(pythonVersion string, plat Platform) *TagSet {
	major := pythonVersion[0:1]
	minor := ""
	if len(pythonVersion) > 1 {
		minor = pythonVersion[1:]
	}

	cpython := "cp" + major + minor
	pyTag := "py" + major

	var tags []Tag
	platformTag := plat.PythonPlatformTag()

	// Exact CPython match
	tags = append(tags, Tag{cpython, cpython, platformTag})
	tags = append(tags, Tag{cpython, "abi3", platformTag})
	tags = append(tags, Tag{cpython, "none", platformTag})

	// Pure Python tags
	tags = append(tags, Tag{cpython, "none", "any"})
	tags = append(tags, Tag{pyTag, "none", platformTag})
	tags = append(tags, Tag{pyTag, "none", "any"})

	// Add manylinux variants for Linux
	if plat.OS == Linux {
		arch := plat.Arch.String()
		for _, ml := range ManylinuxVersions() {
			mlTag := fmt.Sprintf("%s_%s", ml, arch)
			tags = append(tags, Tag{cpython, cpython, mlTag})
			tags = append(tags, Tag{cpython, "abi3", mlTag})
			tags = append(tags, Tag{cpython, "none", mlTag})
		}
		// linux generic
		linuxTag := "linux_" + arch
		tags = append(tags, Tag{cpython, cpython, linuxTag})
		tags = append(tags, Tag{cpython, "abi3", linuxTag})
		tags = append(tags, Tag{cpython, "none", linuxTag})
	}

	// Add macOS variants
	if plat.OS == Darwin {
		arch := plat.PlatformMachine()
		for _, mv := range MacOSVersions(14, 0) {
			macTag := fmt.Sprintf("%s_%s", mv, arch)
			tags = append(tags, Tag{cpython, cpython, macTag})
			tags = append(tags, Tag{cpython, "abi3", macTag})
			tags = append(tags, Tag{cpython, "none", macTag})
		}
		// Universal tags
		for _, mv := range MacOSVersions(14, 0) {
			tags = append(tags, Tag{cpython, cpython, mv + "_universal2"})
			tags = append(tags, Tag{cpython, "abi3", mv + "_universal2"})
			tags = append(tags, Tag{cpython, "none", mv + "_universal2"})
		}
	}

	// Build priority map
	priority := make(map[string]int, len(tags))
	for i, t := range tags {
		key := t.String()
		if _, exists := priority[key]; !exists {
			priority[key] = i
		}
	}

	return &TagSet{Tags: tags, priority: priority}
}

// ParseWheelTags parses the tags from a wheel filename's tag section.
// A wheel tag section looks like: cp311-cp311-manylinux_2_17_x86_64.manylinux2014_x86_64
func ParseWheelTags(python, abi, platform string) []Tag {
	pythons := splitTag(python)
	abis := splitTag(abi)
	platforms := splitTag(platform)

	var tags []Tag
	for _, py := range pythons {
		for _, ab := range abis {
			for _, pl := range platforms {
				tags = append(tags, Tag{Python: py, Abi: ab, Platform: NormalizePlatformTag(pl)})
			}
		}
	}
	return tags
}

func splitTag(s string) []string {
	var parts []string
	current := ""
	for _, ch := range s {
		if ch == '.' {
			if current != "" {
				parts = append(parts, current)
			}
			current = ""
		} else {
			current += string(ch)
		}
	}
	if current != "" {
		parts = append(parts, current)
	}
	return parts
}

// SortByPriority sorts tags by their priority (lower = better).
func SortByPriority(tags []Tag, ts *TagSet) {
	sort.Slice(tags, func(i, j int) bool {
		pi, oki := ts.priority[tags[i].String()]
		pj, okj := ts.priority[tags[j].String()]
		if !oki {
			return false
		}
		if !okj {
			return true
		}
		return pi < pj
	})
}
