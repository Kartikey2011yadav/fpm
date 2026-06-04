package python

import (
	"fmt"
	"path/filepath"
	"runtime"

	"github.com/kartikeyyadav/fpm/internal/pep440"
)

type Interpreter struct {
	Path         string
	Version      pep440.Version
	Prefix       string
	SitePackages string
	SysPaths     []string
	Scheme       InstallScheme
	IsVenv       bool
	IsManaged    bool
	Impl         Implementation
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
	PureLib  string
	PlatLib  string
	Scripts  string
	Data     string
	Include  string
	Headers  string
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
