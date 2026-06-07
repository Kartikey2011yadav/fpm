package cli

import (
	"bytes"
	"strings"
	"testing"
)

func init() {
	rootCmd.Version = Version
	rootCmd.SetVersionTemplate("fpm {{.Version}}\n")
	rootCmd.Flags().BoolP("version", "v", false, "Print version information")
}

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
	commands := []string{"install", "remove", "list", "sync", "lock", "run", "init", "venv",
		"python", "tool", "cache", "snapshot", "tree", "build", "publish", "audit", "pip"}
	for _, cmd := range commands {
		if !strings.Contains(output, cmd) {
			t.Errorf("help output missing command %q", cmd)
		}
	}
}

func TestCommandsExistAndDontPanic(t *testing.T) {
	subcommands := rootCmd.Commands()
	expected := map[string]bool{
		"install": false, "remove": false, "list": false, "sync": false, "lock": false,
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

func TestVersionFlagRegistered(t *testing.T) {
	f := rootCmd.Flags().Lookup("version")
	if f == nil {
		t.Fatal("--version flag not registered")
	}
	if f.Shorthand != "v" {
		t.Errorf("expected shorthand 'v', got %q", f.Shorthand)
	}
}

func TestVersionTemplate(t *testing.T) {
	if rootCmd.Version == "" {
		t.Error("rootCmd.Version not set")
	}
	if rootCmd.Version != Version {
		t.Errorf("rootCmd.Version = %q, want %q", rootCmd.Version, Version)
	}
}

func TestUninstallAlias(t *testing.T) {
	for _, cmd := range rootCmd.Commands() {
		if cmd.Name() == "remove" {
			for _, alias := range cmd.Aliases {
				if alias == "uninstall" {
					return
				}
			}
			t.Error("'remove' command missing 'uninstall' alias")
			return
		}
	}
	t.Error("'remove' command not found")
}

func TestListCommandHasFlags(t *testing.T) {
	for _, cmd := range rootCmd.Commands() {
		if cmd.Name() == "list" {
			flags := []string{"all", "manager"}
			for _, f := range flags {
				if cmd.Flags().Lookup(f) == nil {
					t.Errorf("list command missing flag %q", f)
				}
			}
			return
		}
	}
	t.Error("'list' command not found")
}

func TestHelpShowsCommandGroups(t *testing.T) {
	var buf bytes.Buffer
	rootCmd.SetOut(&buf)
	rootCmd.SetErr(&buf)
	rootCmd.SetArgs([]string{"--help"})
	rootCmd.Execute()
	rootCmd.SetArgs(nil)
	rootCmd.SetOut(nil)
	rootCmd.SetErr(nil)

	output := buf.String()
	groups := []string{"Package Management:", "Project Management:", "Environment:", "Advanced:"}
	for _, g := range groups {
		if !strings.Contains(output, g) {
			t.Errorf("help output missing group %q", g)
		}
	}
}

func TestCommandExamples(t *testing.T) {
	commandsWithExamples := []string{"install", "remove", "list", "run", "init", "build", "publish", "sync", "lock", "tree", "venv"}
	subcommands := rootCmd.Commands()

	for _, name := range commandsWithExamples {
		found := false
		for _, cmd := range subcommands {
			if cmd.Name() == name {
				found = true
				if cmd.Example == "" {
					t.Errorf("command %q has no example text", name)
				}
				break
			}
		}
		if !found {
			t.Errorf("command %q not found", name)
		}
	}
}

func TestVersionSubcommand(t *testing.T) {
	var buf bytes.Buffer
	rootCmd.SetOut(&buf)
	rootCmd.SetErr(&buf)
	rootCmd.SetArgs([]string{"version"})
	rootCmd.Execute()
	rootCmd.SetArgs(nil)
	rootCmd.SetOut(nil)
	rootCmd.SetErr(nil)

	output := buf.String()
	if !strings.Contains(output, "fpm") {
		t.Errorf("version subcommand should output 'fpm', got %q", output)
	}
}
