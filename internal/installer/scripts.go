package installer

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

func InstallScripts(distInfoDir, binDir, pythonPath string) (int, error) {
	entryPointsPath := filepath.Join(distInfoDir, "entry_points.txt")
	scripts, err := parseEntryPoints(entryPointsPath)
	if err != nil {
		return 0, nil // No entry points is not an error
	}

	os.MkdirAll(binDir, 0755)
	installed := 0

	for _, script := range scripts {
		scriptPath := filepath.Join(binDir, script.Name)
		if runtime.GOOS == "windows" {
			scriptPath += ".exe"
		}

		content := generateScriptWrapper(script, pythonPath)
		if err := os.WriteFile(scriptPath, []byte(content), 0755); err != nil {
			continue
		}
		installed++
	}

	return installed, nil
}

func parseEntryPoints(path string) ([]ConsoleScript, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var scripts []ConsoleScript
	inConsoleScripts := false

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		if line == "[console_scripts]" {
			inConsoleScripts = true
			continue
		}
		if strings.HasPrefix(line, "[") {
			inConsoleScripts = false
			continue
		}
		if !inConsoleScripts || line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		// Format: name = module:function
		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}
		name := strings.TrimSpace(parts[0])
		target := strings.TrimSpace(parts[1])

		modFunc := strings.SplitN(target, ":", 2)
		module := modFunc[0]
		funcName := ""
		if len(modFunc) == 2 {
			funcName = modFunc[1]
		}

		scripts = append(scripts, ConsoleScript{
			Name:   name,
			Module: module,
			Func:   funcName,
		})
	}

	return scripts, nil
}

func generateScriptWrapper(script ConsoleScript, pythonPath string) string {
	if runtime.GOOS == "windows" {
		return generateWindowsScript(script, pythonPath)
	}
	return generateUnixScript(script, pythonPath)
}

func generateUnixScript(script ConsoleScript, pythonPath string) string {
	if script.Func != "" {
		return fmt.Sprintf(`#!/bin/sh
'''exec' "%s" "$0" "$@"
' '''
import sys
from %s import %s
sys.exit(%s())
`, pythonPath, script.Module, script.Func, script.Func)
	}
	return fmt.Sprintf(`#!/bin/sh
exec "%s" -m %s "$@"
`, pythonPath, script.Module)
}

func generateWindowsScript(script ConsoleScript, pythonPath string) string {
	if script.Func != "" {
		return fmt.Sprintf(`@echo off
"%s" -c "from %s import %s; %s()" %%*
`, pythonPath, script.Module, script.Func, script.Func)
	}
	return fmt.Sprintf(`@echo off
"%s" -m %s %%*
`, pythonPath, script.Module)
}
