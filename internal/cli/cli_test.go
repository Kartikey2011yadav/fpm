package cli

import (
	"bytes"
	"strings"
	"testing"
)

func TestHelpContainsAllCommands(t *testing.T) {
	var buf bytes.Buffer
	rootCmd.SetOut(&buf)
	rootCmd.SetErr(&buf)
	rootCmd.SetArgs([]string{"--help"})
	rootCmd.Execute()
	rootCmd.SetArgs(nil)
	rootCmd.SetOut(nil)
	rootCmd.SetErr(nil)

	output := buf.String()
	commands := []string{"install", "remove", "sync", "lock", "run", "init", "venv",
		"python", "tool", "cache", "snapshot", "tree", "build", "publish", "audit", "pip"}
	for _, cmd := range commands {
		if !strings.Contains(output, cmd) {
			t.Errorf("help output missing command %q", cmd)
		}
	}
}

func TestCommandsExistAndDontPanic(t *testing.T) {
	// Verify all subcommands are registered and accessible
	subcommands := rootCmd.Commands()
	expected := map[string]bool{
		"install": false, "remove": false, "sync": false, "lock": false,
		"run": false, "init": false, "venv": false, "python": false,
		"tool": false, "cache": false, "snapshot": false, "tree": false,
		"build": false, "publish": false, "audit": false, "pip": false,
		"self": false, "version": false,
	}

	for _, cmd := range subcommands {
		expected[cmd.Name()] = true
	}

	for name, found := range expected {
		if !found {
			t.Errorf("command %q not registered", name)
		}
	}
}
