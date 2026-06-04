package pep508

import (
	"github.com/kartikeyyadav/fpm/internal/pep440"
	"github.com/kartikeyyadav/fpm/pkg/types"
)

type Requirement struct {
	Name       types.PackageName
	Extras     []types.ExtraName
	Specifiers pep440.VersionSpecifiers
	URL        string
	Marker     MarkerTree
}

func (r Requirement) String() string {
	s := r.Name.Raw()

	if len(r.Extras) > 0 {
		s += "["
		for i, e := range r.Extras {
			if i > 0 {
				s += ","
			}
			s += e.Raw()
		}
		s += "]"
	}

	if r.URL != "" {
		s += " @ " + r.URL
	} else if len(r.Specifiers) > 0 {
		s += " " + r.Specifiers.String()
	}

	if r.Marker != nil {
		s += " ; " + r.Marker.String()
	}

	return s
}

func (r Requirement) MatchesVersion(v pep440.Version) bool {
	return r.Specifiers.Contains(v)
}

func (r Requirement) EvaluateMarkers(env MarkerEnvironment) bool {
	if r.Marker == nil {
		return true
	}
	return r.Marker.Evaluate(env)
}
