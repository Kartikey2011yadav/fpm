package build

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

type BuildFrontend struct {
	PythonPath string
	SourceDir  string
	OutDir     string
}

type BuildResult struct {
	WheelPath string
	SdistPath string
}

func NewFrontend(pythonPath, sourceDir, outDir string) *BuildFrontend {
	return &BuildFrontend{
		PythonPath: pythonPath,
		SourceDir:  sourceDir,
		OutDir:     outDir,
	}
}

func (bf *BuildFrontend) BuildWheel() (*BuildResult, error) {
	if err := os.MkdirAll(bf.OutDir, 0755); err != nil {
		return nil, err
	}

	// PEP 517: invoke build backend via python -m build
	script := `
import sys
import os
sys.path.insert(0, os.getcwd())

try:
    from build import ProjectBuilder
    builder = ProjectBuilder(os.getcwd())
    result = builder.build('wheel', output_directory=sys.argv[1])
    print(result)
except ImportError:
    # Fallback: use setuptools directly
    import subprocess
    result = subprocess.run(
        [sys.executable, 'setup.py', 'bdist_wheel', '--dist-dir', sys.argv[1]],
        capture_output=True, text=True
    )
    if result.returncode != 0:
        print(result.stderr, file=sys.stderr)
        sys.exit(1)
    # Find the wheel
    import glob
    wheels = glob.glob(os.path.join(sys.argv[1], '*.whl'))
    if wheels:
        print(wheels[0])
    else:
        sys.exit(1)
`

	cmd := exec.Command(bf.PythonPath, "-c", script, bf.OutDir)
	cmd.Dir = bf.SourceDir

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("wheel build failed: %s\n%s", err, stderr.String())
	}

	wheelPath := filepath.Clean(stdout.String())
	return &BuildResult{WheelPath: wheelPath}, nil
}

func (bf *BuildFrontend) BuildSdist() (*BuildResult, error) {
	if err := os.MkdirAll(bf.OutDir, 0755); err != nil {
		return nil, err
	}

	script := `
import sys
import os
sys.path.insert(0, os.getcwd())

try:
    from build import ProjectBuilder
    builder = ProjectBuilder(os.getcwd())
    result = builder.build('sdist', output_directory=sys.argv[1])
    print(result)
except ImportError:
    import subprocess
    result = subprocess.run(
        [sys.executable, 'setup.py', 'sdist', '--dist-dir', sys.argv[1]],
        capture_output=True, text=True
    )
    if result.returncode != 0:
        print(result.stderr, file=sys.stderr)
        sys.exit(1)
    import glob
    sdists = glob.glob(os.path.join(sys.argv[1], '*.tar.gz'))
    if sdists:
        print(sdists[0])
    else:
        sys.exit(1)
`

	cmd := exec.Command(bf.PythonPath, "-c", script, bf.OutDir)
	cmd.Dir = bf.SourceDir

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("sdist build failed: %s\n%s", err, stderr.String())
	}

	sdistPath := filepath.Clean(stdout.String())
	return &BuildResult{SdistPath: sdistPath}, nil
}
