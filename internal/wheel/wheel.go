package wheel

import (
	"archive/zip"
	"bufio"
	"fmt"
	"strings"

	"github.com/kartikeyyadav/fpm/internal/pep440"
	"github.com/kartikeyyadav/fpm/internal/platform"
	"github.com/kartikeyyadav/fpm/pkg/types"
)

type WheelFilename struct {
	Name      types.PackageName
	Version   pep440.Version
	BuildTag  string
	PythonTag []string
	AbiTag    []string
	Platform  []string
}

func (w WheelFilename) String() string {
	parts := []string{
		w.Name.Normalized(),
		w.Version.String(),
	}
	if w.BuildTag != "" {
		parts = append(parts, w.BuildTag)
	}
	parts = append(parts,
		strings.Join(w.PythonTag, "."),
		strings.Join(w.AbiTag, "."),
		strings.Join(w.Platform, "."),
	)
	return strings.Join(parts, "-") + ".whl"
}

func (w WheelFilename) Tags() []platform.Tag {
	return platform.ParseWheelTags(
		strings.Join(w.PythonTag, "."),
		strings.Join(w.AbiTag, "."),
		strings.Join(w.Platform, "."),
	)
}

func (w WheelFilename) IsPureWheel() bool {
	for _, p := range w.Platform {
		if p == "any" {
			return true
		}
	}
	return false
}

func ParseFilename(filename string) (*WheelFilename, error) {
	if !strings.HasSuffix(filename, ".whl") {
		return nil, fmt.Errorf("not a wheel filename: %q", filename)
	}
	filename = strings.TrimSuffix(filename, ".whl")

	parts := strings.Split(filename, "-")
	if len(parts) < 5 || len(parts) > 6 {
		return nil, fmt.Errorf("invalid wheel filename (expected 5-6 parts): %q", filename)
	}

	var w WheelFilename
	w.Name = types.NewPackageName(parts[0])

	ver, err := pep440.Parse(parts[1])
	if err != nil {
		return nil, fmt.Errorf("invalid version in wheel filename: %w", err)
	}
	w.Version = ver

	idx := 2
	if len(parts) == 6 {
		w.BuildTag = parts[idx]
		idx++
	}

	w.PythonTag = strings.Split(parts[idx], ".")
	w.AbiTag = strings.Split(parts[idx+1], ".")
	w.Platform = strings.Split(parts[idx+2], ".")

	return &w, nil
}

type Metadata struct {
	Name           string
	Version        string
	Summary        string
	RequiresPython string
	RequiresDist   []string
	Provides       []string
}

func ReadMetadataFromZip(wheelPath string) (*Metadata, error) {
	r, err := zip.OpenReader(wheelPath)
	if err != nil {
		return nil, err
	}
	defer r.Close()

	for _, f := range r.File {
		if strings.HasSuffix(f.Name, ".dist-info/METADATA") {
			rc, err := f.Open()
			if err != nil {
				return nil, err
			}
			defer rc.Close()

			return parseMetadata(rc)
		}
	}

	return nil, fmt.Errorf("METADATA not found in wheel")
}

func parseMetadata(r interface{ Read([]byte) (int, error) }) (*Metadata, error) {
	m := &Metadata{}
	scanner := bufio.NewScanner(r)

	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			break // End of headers
		}

		if strings.HasPrefix(line, "Name: ") {
			m.Name = strings.TrimPrefix(line, "Name: ")
		} else if strings.HasPrefix(line, "Version: ") {
			m.Version = strings.TrimPrefix(line, "Version: ")
		} else if strings.HasPrefix(line, "Summary: ") {
			m.Summary = strings.TrimPrefix(line, "Summary: ")
		} else if strings.HasPrefix(line, "Requires-Python: ") {
			m.RequiresPython = strings.TrimPrefix(line, "Requires-Python: ")
		} else if strings.HasPrefix(line, "Requires-Dist: ") {
			m.RequiresDist = append(m.RequiresDist, strings.TrimPrefix(line, "Requires-Dist: "))
		} else if strings.HasPrefix(line, "Provides-Extra: ") {
			m.Provides = append(m.Provides, strings.TrimPrefix(line, "Provides-Extra: "))
		}
	}

	return m, scanner.Err()
}
