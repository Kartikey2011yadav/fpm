package venv

import (
	"fmt"
	"os"
	"path/filepath"
)

func writeActivationScripts(binDir, venvPath, prompt string) error {
	scripts := map[string]string{
		"activate":       bashActivateScript(venvPath, binDir, prompt),
		"activate.fish":  fishActivateScript(venvPath, binDir, prompt),
		"activate.csh":   cshActivateScript(venvPath, binDir, prompt),
		"activate.ps1":   powershellActivateScript(venvPath, binDir, prompt),
		"activate.bat":   batActivateScript(venvPath, binDir, prompt),
		"deactivate.bat": batDeactivateScript(),
	}

	for name, content := range scripts {
		path := filepath.Join(binDir, name)
		perm := os.FileMode(0644)
		if name == "activate" || name == "activate.fish" || name == "activate.csh" {
			perm = 0755
		}
		if err := os.WriteFile(path, []byte(content), perm); err != nil {
			return err
		}
	}
	return nil
}

func bashActivateScript(venvPath, binDir, prompt string) string {
	return fmt.Sprintf(`# This file must be used with "source bin/activate" *from bash*
# You cannot run it directly

deactivate () {
    if [ -n "${_OLD_VIRTUAL_PATH:-}" ] ; then
        PATH="${_OLD_VIRTUAL_PATH:-}"
        export PATH
        unset _OLD_VIRTUAL_PATH
    fi
    if [ -n "${_OLD_VIRTUAL_PS1:-}" ] ; then
        PS1="${_OLD_VIRTUAL_PS1:-}"
        export PS1
        unset _OLD_VIRTUAL_PS1
    fi
    unset VIRTUAL_ENV
    unset VIRTUAL_ENV_PROMPT
    if [ ! "${1:-}" = "nondestructive" ] ; then
        unset -f deactivate
    fi
}

# unset irrelevant variables
deactivate nondestructive

VIRTUAL_ENV="%s"
export VIRTUAL_ENV

VIRTUAL_ENV_PROMPT="%s"
export VIRTUAL_ENV_PROMPT

_OLD_VIRTUAL_PATH="$PATH"
PATH="%s:$PATH"
export PATH

_OLD_VIRTUAL_PS1="${PS1:-}"
PS1="(%s) ${PS1:-}"
export PS1
`, venvPath, prompt, binDir, prompt)
}

func fishActivateScript(venvPath, binDir, prompt string) string {
	return fmt.Sprintf(`# This file must be used with "source bin/activate.fish" *from fish*

function deactivate -d "Exit virtual environment"
    if test -n "$_OLD_VIRTUAL_PATH"
        set -gx PATH $_OLD_VIRTUAL_PATH
        set -e _OLD_VIRTUAL_PATH
    end
    if test -n "$_OLD_FISH_PROMPT_OVERRIDE"
        set -e _OLD_FISH_PROMPT_OVERRIDE
        functions -e fish_prompt
        if functions -q _old_fish_prompt
            functions -c _old_fish_prompt fish_prompt
            functions -e _old_fish_prompt
        end
    end
    set -e VIRTUAL_ENV
    set -e VIRTUAL_ENV_PROMPT
end

deactivate

set -gx VIRTUAL_ENV "%s"
set -gx VIRTUAL_ENV_PROMPT "%s"
set -gx _OLD_VIRTUAL_PATH $PATH
set -gx PATH "%s" $PATH

set -gx _OLD_FISH_PROMPT_OVERRIDE "1"
functions -c fish_prompt _old_fish_prompt
function fish_prompt
    printf "(%s) "
    _old_fish_prompt
end
`, venvPath, prompt, binDir, prompt)
}

func cshActivateScript(venvPath, binDir, prompt string) string {
	return fmt.Sprintf(`# This file must be used with "source bin/activate.csh" *from csh*
setenv VIRTUAL_ENV "%s"
setenv VIRTUAL_ENV_PROMPT "%s"
set _OLD_VIRTUAL_PATH="$PATH"
setenv PATH "%s:$PATH"
set prompt="(%s) $prompt"
`, venvPath, prompt, binDir, prompt)
}

func powershellActivateScript(venvPath, binDir, prompt string) string {
	return fmt.Sprintf(`# PowerShell activation script
$script:THIS_PATH = $myinvocation.mycommand.path
$script:BASE_DIR = Split-Path (Resolve-Path "$THIS_PATH/..") -Parent

function global:deactivate ([switch]$NonDestructive) {
    if (Test-Path variable:_OLD_VIRTUAL_PATH) {
        $env:PATH = $variable:_OLD_VIRTUAL_PATH
        Remove-Variable "_OLD_VIRTUAL_PATH" -Scope global
    }
    if (Test-Path function:_OLD_VIRTUAL_PROMPT) {
        Set-Item function:prompt -Value $function:_OLD_VIRTUAL_PROMPT
        Remove-Item function:_OLD_VIRTUAL_PROMPT
    }
    Remove-Item env:VIRTUAL_ENV -ErrorAction SilentlyContinue
    Remove-Item env:VIRTUAL_ENV_PROMPT -ErrorAction SilentlyContinue
    if (-not $NonDestructive) {
        Remove-Item function:deactivate
    }
}

deactivate -NonDestructive

$env:VIRTUAL_ENV = "%s"
$env:VIRTUAL_ENV_PROMPT = "%s"

$_OLD_VIRTUAL_PATH = $env:PATH
$env:PATH = "%s" + [IO.Path]::PathSeparator + $env:PATH

Copy-Item function:prompt function:_OLD_VIRTUAL_PROMPT
function global:prompt {
    "(%s) $(& $_OLD_VIRTUAL_PROMPT)"
}
`, venvPath, prompt, binDir, prompt)
}

func batActivateScript(venvPath, binDir, prompt string) string {
	return fmt.Sprintf(`@echo off
set "VIRTUAL_ENV=%s"
set "VIRTUAL_ENV_PROMPT=%s"
set "_OLD_VIRTUAL_PATH=%%PATH%%"
set "PATH=%s;%%PATH%%"
set "_OLD_VIRTUAL_PROMPT=%%PROMPT%%"
set "PROMPT=(%s) %%PROMPT%%"
`, venvPath, prompt, binDir, prompt)
}

func batDeactivateScript() string {
	return `@echo off
if defined _OLD_VIRTUAL_PATH (
    set "PATH=%_OLD_VIRTUAL_PATH%"
    set "_OLD_VIRTUAL_PATH="
)
if defined _OLD_VIRTUAL_PROMPT (
    set "PROMPT=%_OLD_VIRTUAL_PROMPT%"
    set "_OLD_VIRTUAL_PROMPT="
)
set "VIRTUAL_ENV="
set "VIRTUAL_ENV_PROMPT="
`
}
