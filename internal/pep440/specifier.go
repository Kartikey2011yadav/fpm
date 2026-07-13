package pep440

import (
	"fmt"
	"regexp"
	"strings"
)

type Operator int

const (
	OpEqual            Operator = iota // ==
	OpNotEqual                         // !=
	OpLessThan                         // <
	OpLessThanEqual                    // <=
	OpGreaterThan                      // >
	OpGreaterThanEqual                 // >=
	OpCompatible                       // ~=
	OpArbitrary                        // ===
)

func (op Operator) String() string {
	switch op {
	case OpEqual:
		return "=="
	case OpNotEqual:
		return "!="
	case OpLessThan:
		return "<"
	case OpLessThanEqual:
		return "<="
	case OpGreaterThan:
		return ">"
	case OpGreaterThanEqual:
		return ">="
	case OpCompatible:
		return "~="
	case OpArbitrary:
		return "==="
	default:
		return "?"
	}
}

type Specifier struct {
	Op       Operator
	Version  Version
	Wildcard bool
}

func (s Specifier) String() string {
	ver := s.Version.String()
	if s.Wildcard {
		ver += ".*"
	}
	return s.Op.String() + ver
}

func (s Specifier) Contains(v Version) bool {
	switch s.Op {
	case OpEqual:
		return matchEqual(s, v)
	case OpNotEqual:
		return !matchEqual(s, v)
	case OpLessThan:
		return compareIgnoringLocal(v, s.Version) < 0
	case OpLessThanEqual:
		return compareIgnoringLocal(v, s.Version) <= 0
	case OpGreaterThan:
		return compareIgnoringLocal(v, s.Version) > 0
	case OpGreaterThanEqual:
		return compareIgnoringLocal(v, s.Version) >= 0
	case OpCompatible:
		return matchCompatible(s, v)
	case OpArbitrary:
		return v.String() == s.Version.String()
	default:
		return false
	}
}

// compareIgnoringLocal compares versions ignoring local segments per PEP 440.
func compareIgnoringLocal(a, b Version) int {
	aCopy := a
	bCopy := b
	aCopy.Local = nil
	bCopy.Local = nil
	return Compare(aCopy, bCopy)
}

func matchEqual(s Specifier, v Version) bool {
	if s.Wildcard {
		// == 1.2.* means any version starting with 1.2
		rel := s.Version.Release
		if len(v.Release) < len(rel) {
			return false
		}
		for i, seg := range rel {
			if v.Release[i] != seg {
				return false
			}
		}
		return v.Epoch == s.Version.Epoch
	}
	// PEP 440: for == (non-===), ignore local version on the candidate
	// if the specifier has no local segment. E.g. ==1.0 matches 1.0+local1.
	candidate := v
	if len(s.Version.Local) == 0 {
		candidate.Local = nil
	}
	return Compare(candidate, s.Version) == 0
}

func matchCompatible(s Specifier, v Version) bool {
	// ~= X.Y is equivalent to >= X.Y, == X.*
	if Compare(v, s.Version) < 0 {
		return false
	}
	// Must match the release prefix (all segments except last)
	rel := s.Version.Release
	if len(rel) < 2 {
		return Compare(v, s.Version) >= 0
	}
	prefix := rel[:len(rel)-1]
	if len(v.Release) < len(prefix) {
		return false
	}
	for i, seg := range prefix {
		if v.Release[i] != seg {
			return false
		}
	}
	return v.Epoch == s.Version.Epoch
}

type VersionSpecifiers []Specifier

func (vs VersionSpecifiers) Contains(v Version) bool {
	for _, s := range vs {
		if !s.Contains(v) {
			return false
		}
	}
	return true
}

func (vs VersionSpecifiers) String() string {
	parts := make([]string, len(vs))
	for i, s := range vs {
		parts[i] = s.String()
	}
	return strings.Join(parts, ", ")
}

var specRegex = regexp.MustCompile(`^\s*(~=|===|==|!=|<=|>=|<|>)\s*(.+?)\s*$`)

func ParseSpecifier(input string) (Specifier, error) {
	m := specRegex.FindStringSubmatch(input)
	if m == nil {
		return Specifier{}, fmt.Errorf("invalid specifier: %q", input)
	}

	opStr := m[1]
	verStr := m[2]

	var op Operator
	switch opStr {
	case "==":
		op = OpEqual
	case "!=":
		op = OpNotEqual
	case "<":
		op = OpLessThan
	case "<=":
		op = OpLessThanEqual
	case ">":
		op = OpGreaterThan
	case ">=":
		op = OpGreaterThanEqual
	case "~=":
		op = OpCompatible
	case "===":
		op = OpArbitrary
	}

	wildcard := false
	if strings.HasSuffix(verStr, ".*") {
		wildcard = true
		verStr = strings.TrimSuffix(verStr, ".*")
	}

	ver, err := Parse(verStr)
	if err != nil {
		if op == OpArbitrary {
			ver = Version{Release: []int{0}}
		} else {
			return Specifier{}, fmt.Errorf("invalid version in specifier %q: %w", input, err)
		}
	}

	return Specifier{Op: op, Version: ver, Wildcard: wildcard}, nil
}

func ParseSpecifiers(input string) (VersionSpecifiers, error) {
	input = strings.TrimSpace(input)
	if input == "" {
		return nil, nil
	}

	parts := strings.Split(input, ",")
	specs := make(VersionSpecifiers, 0, len(parts))

	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		s, err := ParseSpecifier(p)
		if err != nil {
			return nil, err
		}
		specs = append(specs, s)
	}

	return specs, nil
}
