package pep440

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

var versionRegex = regexp.MustCompile(
	`(?i)^\s*v?` +
		`(?:(?P<epoch>[0-9]+)!)?` +
		`(?P<release>[0-9]+(?:\.[0-9]+)*)` +
		`(?P<pre>[-_.]?(?:alpha|a|beta|b|preview|c|rc)[-_.]?[0-9]*)?` +
		`(?P<post>(?:-[0-9]+)|(?:[-_.]?(?:post|rev|r)[-_.]?[0-9]*))?` +
		`(?P<dev>[-_.]?dev[-_.]?[0-9]*)?` +
		`(?:\+(?P<local>[a-z0-9]+(?:[-_.][a-z0-9]+)*))?` +
		`\s*$`,
)

func Parse(input string) (Version, error) {
	input = strings.TrimSpace(input)
	if input == "" {
		return Version{}, fmt.Errorf("empty version string")
	}

	matches := versionRegex.FindStringSubmatch(strings.ToLower(input))
	if matches == nil {
		return Version{}, fmt.Errorf("invalid version: %q", input)
	}

	var v Version

	epochStr := matches[versionRegex.SubexpIndex("epoch")]
	if epochStr != "" {
		epoch, err := strconv.Atoi(epochStr)
		if err != nil {
			return Version{}, fmt.Errorf("invalid epoch: %q", epochStr)
		}
		v.Epoch = epoch
	}

	releaseStr := matches[versionRegex.SubexpIndex("release")]
	parts := strings.Split(releaseStr, ".")
	v.Release = make([]int, len(parts))
	for i, p := range parts {
		n, err := strconv.Atoi(p)
		if err != nil {
			return Version{}, fmt.Errorf("invalid release segment: %q", p)
		}
		v.Release[i] = n
	}

	preStr := matches[versionRegex.SubexpIndex("pre")]
	if preStr != "" {
		pre, err := parsePre(preStr)
		if err != nil {
			return Version{}, err
		}
		v.Pre = pre
	}

	postStr := matches[versionRegex.SubexpIndex("post")]
	if postStr != "" {
		post, err := parsePost(postStr)
		if err != nil {
			return Version{}, err
		}
		v.Post = post
	}

	devStr := matches[versionRegex.SubexpIndex("dev")]
	if devStr != "" {
		dev, err := parseDev(devStr)
		if err != nil {
			return Version{}, err
		}
		v.Dev = dev
	}

	localStr := matches[versionRegex.SubexpIndex("local")]
	if localStr != "" {
		v.Local = parseLocal(localStr)
	}

	return v, nil
}

func parsePre(s string) (*PreRelease, error) {
	s = strings.TrimLeft(s, "-_.")
	s = strings.ToLower(s)

	var kind PreReleaseKind
	var numStr string

	switch {
	case strings.HasPrefix(s, "alpha"):
		kind = PreAlpha
		numStr = strings.TrimPrefix(s, "alpha")
	case strings.HasPrefix(s, "a"):
		kind = PreAlpha
		numStr = strings.TrimPrefix(s, "a")
	case strings.HasPrefix(s, "beta"):
		kind = PreBeta
		numStr = strings.TrimPrefix(s, "beta")
	case strings.HasPrefix(s, "b"):
		kind = PreBeta
		numStr = strings.TrimPrefix(s, "b")
	case strings.HasPrefix(s, "preview"):
		kind = PreRC
		numStr = strings.TrimPrefix(s, "preview")
	case strings.HasPrefix(s, "rc"):
		kind = PreRC
		numStr = strings.TrimPrefix(s, "rc")
	case strings.HasPrefix(s, "c"):
		kind = PreRC
		numStr = strings.TrimPrefix(s, "c")
	default:
		return nil, fmt.Errorf("invalid pre-release: %q", s)
	}

	numStr = strings.TrimLeft(numStr, "-_.")
	num := 0
	if numStr != "" {
		var err error
		num, err = strconv.Atoi(numStr)
		if err != nil {
			return nil, fmt.Errorf("invalid pre-release number: %q", numStr)
		}
	}

	return &PreRelease{Kind: kind, Number: num}, nil
}

func parsePost(s string) (*int, error) {
	s = strings.TrimLeft(s, "-_.")
	s = strings.ToLower(s)

	var numStr string
	switch {
	case strings.HasPrefix(s, "post"):
		numStr = strings.TrimPrefix(s, "post")
	case strings.HasPrefix(s, "rev"):
		numStr = strings.TrimPrefix(s, "rev")
	case strings.HasPrefix(s, "r"):
		numStr = strings.TrimPrefix(s, "r")
	default:
		numStr = s
	}

	numStr = strings.TrimLeft(numStr, "-_.")
	num := 0
	if numStr != "" {
		var err error
		num, err = strconv.Atoi(numStr)
		if err != nil {
			return nil, fmt.Errorf("invalid post-release number: %q", numStr)
		}
	}

	return &num, nil
}

func parseDev(s string) (*int, error) {
	s = strings.TrimLeft(s, "-_.")
	s = strings.ToLower(s)
	s = strings.TrimPrefix(s, "dev")
	s = strings.TrimLeft(s, "-_.")

	num := 0
	if s != "" {
		var err error
		num, err = strconv.Atoi(s)
		if err != nil {
			return nil, fmt.Errorf("invalid dev-release number: %q", s)
		}
	}

	return &num, nil
}

func parseLocal(s string) []string {
	s = strings.ToLower(s)
	sep := func(r rune) bool { return r == '-' || r == '_' || r == '.' }
	return strings.FieldsFunc(s, sep)
}
