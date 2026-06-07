package python

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"runtime"

	"github.com/kartikeyyadav/fpm/internal/pep440"
)

type Interpreter struct {
	Path         string        `json:"path"`
	Version      pep440.Version `json:"-"`
	VersionStr   string        `json:"version"`
	Prefix       string        `json:"prefix"`
	SitePackages string        `json:"site_packages"`
	SysPaths     []string      `json:"sys_paths"`
	Scheme       InstallScheme `json:"scheme"`
	IsVenv       bool          `json:"is_venv"`
	IsManaged    bool          `json:"is_managed"`
	Impl         Implementation `json:"impl"`
}

func (i *Interpreter) MarshalJSON() ([]byte, error) {
	type Alias Interpreter
	return json.Marshal(&struct {
		*Alias
		VersionStr string `json:"version"`
	}{
		Alias:      (*Alias)(i),
		VersionStr: i.Version.String(),
	})
}

func (i *Interpreter) UnmarshalJSON(data []byte) error {
	type Alias Interpreter
	aux := &struct {
		*Alias
		VersionStr string `json:"version"`
	}{
		Alias: (*Alias)(i),
	}
	if err := json.Unmarshal(data, aux); err != nil {
		return err
	}
	if aux.VersionStr != "" {
		v, err := pep440.Parse(aux.VersionStr)
		if err != nil {
			return err
		}
		i.Version = v
		i.VersionStr = aux.VersionStr
	}
	return nil
}

type Implementation int

const (
	CPython Implementation = iota
	PyPy
	GraalPy
)

func (i Implementation) String() string {
	switch i {
	case CPython:
		return "cpython"
	case PyPy:
		return "pypy"
	case GraalPy:
		return "graalpy"
	default:
		return "unknown"
	}
}

type InstallScheme struct {
	PureLib  string `json:"purelib"`
	PlatLib  string `json:"platlib"`
	Scripts  string `json:"scripts"`
	Data     string `json:"data"`
	Include  string `json:"include"`
	Headers  string `json:"headers"`
}

func (i *Interpreter) PythonTag() string {
	prefix := "cp"
	if i.Impl == PyPy {
		prefix = "pp"
	}
	return fmt.Sprintf("%s%d%d", prefix, i.Version.Major(), i.Version.Minor())
}

func (i *Interpreter) BinDir() string {
	if runtime.GOOS == "windows" {
		return filepath.Join(i.Prefix, "Scripts")
	}
	return filepath.Join(i.Prefix, "bin")
}

func (i *Interpreter) LibDir() string {
	if runtime.GOOS == "windows" {
		return filepath.Join(i.Prefix, "Lib")
	}
	return filepath.Join(i.Prefix, "lib",
		fmt.Sprintf("python%d.%d", i.Version.Major(), i.Version.Minor()))
}

func (i *Interpreter) VersionString() string {
	return fmt.Sprintf("%d.%d.%d", i.Version.Major(), i.Version.Minor(), i.Version.Patch())
}
