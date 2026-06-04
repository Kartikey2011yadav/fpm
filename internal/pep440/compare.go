package pep440

import (
	"strconv"
	"strings"
)

func Compare(a, b Version) int {
	if c := compareInt(a.Epoch, b.Epoch); c != 0 {
		return c
	}

	if c := compareRelease(a.Release, b.Release); c != 0 {
		return c
	}

	// PEP 440 ordering within same release: dev < pre < release < post
	// We need to handle the interaction between Pre, Post, and Dev carefully.
	// A version with dev but no pre (1.0.dev1) is before any pre-release (1.0a1).
	aPhase := versionPhase(a)
	bPhase := versionPhase(b)

	if aPhase != bPhase {
		return compareInt(aPhase, bPhase)
	}

	// Same phase — compare within phase
	if a.Pre != nil && b.Pre != nil {
		if c := compareInt(int(a.Pre.Kind), int(b.Pre.Kind)); c != 0 {
			return c
		}
		if c := compareInt(a.Pre.Number, b.Pre.Number); c != 0 {
			return c
		}
	}

	if a.Post != nil && b.Post != nil {
		if c := compareInt(*a.Post, *b.Post); c != 0 {
			return c
		}
	}

	if a.Dev != nil && b.Dev != nil {
		if c := compareInt(*a.Dev, *b.Dev); c != 0 {
			return c
		}
	}

	return compareLocal(a.Local, b.Local)
}

// versionPhase assigns a numeric phase for ordering:
// dev-only (no pre, no post) = 0
// pre + dev = 1
// pre (no dev) = 2
// release (no pre, no post, no dev) = 3
// post + dev = 4
// post (no dev) = 5
func versionPhase(v Version) int {
	hasPre := v.Pre != nil
	hasPost := v.Post != nil
	hasDev := v.Dev != nil

	switch {
	case !hasPre && !hasPost && hasDev:
		return 0 // dev release only (e.g. 1.0.dev1)
	case hasPre && hasDev:
		return 1 // pre + dev (e.g. 1.0a1.dev1)
	case hasPre && !hasDev:
		return 2 // pre-release (e.g. 1.0a1)
	case !hasPre && !hasPost && !hasDev:
		return 3 // final release (e.g. 1.0)
	case hasPost && hasDev:
		return 4 // post + dev (e.g. 1.0.post1.dev1)
	case hasPost && !hasDev:
		return 5 // post-release (e.g. 1.0.post1)
	default:
		return 3
	}
}

func compareRelease(a, b []int) int {
	maxLen := len(a)
	if len(b) > maxLen {
		maxLen = len(b)
	}

	for i := 0; i < maxLen; i++ {
		av := 0
		if i < len(a) {
			av = a[i]
		}
		bv := 0
		if i < len(b) {
			bv = b[i]
		}
		if av < bv {
			return -1
		}
		if av > bv {
			return 1
		}
	}
	return 0
}

func comparePre(a, b *PreRelease) int {
	// No pre-release > any pre-release (1.0 > 1.0a1)
	if a == nil && b == nil {
		return 0
	}
	if a == nil {
		return 1
	}
	if b == nil {
		return -1
	}

	if a.Kind != b.Kind {
		return int(a.Kind) - int(b.Kind)
	}
	return compareInt(a.Number, b.Number)
}

func comparePost(a, b *int) int {
	// No post-release < any post-release (1.0 < 1.0.post1)
	if a == nil && b == nil {
		return 0
	}
	if a == nil {
		return -1
	}
	if b == nil {
		return 1
	}
	return compareInt(*a, *b)
}

func compareDev(a, b *int) int {
	// No dev release > any dev release (1.0 > 1.0.dev1)
	if a == nil && b == nil {
		return 0
	}
	if a == nil {
		return 1
	}
	if b == nil {
		return -1
	}
	return compareInt(*a, *b)
}

func compareLocal(a, b []string) int {
	// No local < any local (1.0 < 1.0+local)
	if len(a) == 0 && len(b) == 0 {
		return 0
	}
	if len(a) == 0 {
		return -1
	}
	if len(b) == 0 {
		return 1
	}

	maxLen := len(a)
	if len(b) > maxLen {
		maxLen = len(b)
	}

	for i := 0; i < maxLen; i++ {
		var av, bv string
		if i < len(a) {
			av = a[i]
		}
		if i < len(b) {
			bv = b[i]
		}

		aNum, aIsNum := tryParseInt(av)
		bNum, bIsNum := tryParseInt(bv)

		switch {
		case aIsNum && bIsNum:
			if c := compareInt(aNum, bNum); c != 0 {
				return c
			}
		case aIsNum:
			return -1
		case bIsNum:
			return 1
		default:
			if c := strings.Compare(av, bv); c != 0 {
				return c
			}
		}
	}
	return 0
}

func compareInt(a, b int) int {
	if a < b {
		return -1
	}
	if a > b {
		return 1
	}
	return 0
}

func tryParseInt(s string) (int, bool) {
	n, err := strconv.Atoi(s)
	if err != nil {
		return 0, false
	}
	return n, true
}
