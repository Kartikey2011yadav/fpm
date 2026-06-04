package types

import (
	"strings"
	"unicode"
)

type PackageName struct {
	raw        string
	normalized string
}

func NewPackageName(name string) PackageName {
	return PackageName{
		raw:        name,
		normalized: normalizeName(name),
	}
}

func (p PackageName) Raw() string        { return p.raw }
func (p PackageName) Normalized() string  { return p.normalized }
func (p PackageName) String() string      { return p.normalized }
func (p PackageName) IsEmpty() bool       { return p.normalized == "" }

func (p PackageName) Equal(other PackageName) bool {
	return p.normalized == other.normalized
}

func normalizeName(name string) string {
	var b strings.Builder
	b.Grow(len(name))
	for _, r := range name {
		switch {
		case r == '-' || r == '_' || r == '.':
			b.WriteByte('-')
		default:
			b.WriteRune(unicode.ToLower(r))
		}
	}
	return b.String()
}

type ExtraName struct {
	raw        string
	normalized string
}

func NewExtraName(name string) ExtraName {
	return ExtraName{
		raw:        name,
		normalized: normalizeName(name),
	}
}

func (e ExtraName) Raw() string       { return e.raw }
func (e ExtraName) Normalized() string { return e.normalized }
func (e ExtraName) String() string     { return e.normalized }

type HashDigest struct {
	Algorithm string
	Value     string
}

func (h HashDigest) String() string {
	return h.Algorithm + ":" + h.Value
}
