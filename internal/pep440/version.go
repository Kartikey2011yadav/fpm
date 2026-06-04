package pep440

import (
	"fmt"
	"strings"
)

type PreReleaseKind int

const (
	PreAlpha PreReleaseKind = iota
	PreBeta
	PreRC
)

func (k PreReleaseKind) String() string {
	switch k {
	case PreAlpha:
		return "a"
	case PreBeta:
		return "b"
	case PreRC:
		return "rc"
	default:
		return "?"
	}
}

type PreRelease struct {
	Kind   PreReleaseKind
	Number int
}

type Version struct {
	Epoch   int
	Release []int
	Pre     *PreRelease
	Post    *int
	Dev     *int
	Local   []string
}

func (v Version) String() string {
	var b strings.Builder

	if v.Epoch != 0 {
		fmt.Fprintf(&b, "%d!", v.Epoch)
	}

	for i, seg := range v.Release {
		if i > 0 {
			b.WriteByte('.')
		}
		fmt.Fprintf(&b, "%d", seg)
	}

	if v.Pre != nil {
		fmt.Fprintf(&b, "%s%d", v.Pre.Kind, v.Pre.Number)
	}
	if v.Post != nil {
		fmt.Fprintf(&b, ".post%d", *v.Post)
	}
	if v.Dev != nil {
		fmt.Fprintf(&b, ".dev%d", *v.Dev)
	}
	if len(v.Local) > 0 {
		b.WriteByte('+')
		b.WriteString(strings.Join(v.Local, "."))
	}

	return b.String()
}

func (v Version) IsPreRelease() bool {
	return v.Pre != nil || v.Dev != nil
}

func (v Version) IsPostRelease() bool {
	return v.Post != nil
}

func (v Version) IsDevRelease() bool {
	return v.Dev != nil
}

func (v Version) IsLocal() bool {
	return len(v.Local) > 0
}

func (v Version) BaseVersion() Version {
	return Version{
		Epoch:   v.Epoch,
		Release: v.Release,
	}
}

func (v Version) Major() int {
	if len(v.Release) > 0 {
		return v.Release[0]
	}
	return 0
}

func (v Version) Minor() int {
	if len(v.Release) > 1 {
		return v.Release[1]
	}
	return 0
}

func (v Version) Patch() int {
	if len(v.Release) > 2 {
		return v.Release[2]
	}
	return 0
}

func (v Version) Equal(other Version) bool {
	return Compare(v, other) == 0
}

func (v Version) LessThan(other Version) bool {
	return Compare(v, other) < 0
}

func (v Version) GreaterThan(other Version) bool {
	return Compare(v, other) > 0
}
