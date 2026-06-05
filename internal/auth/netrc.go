package auth

import (
	"bufio"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

type NetrcEntry struct {
	Machine  string
	Login    string
	Password string
}

func LookupNetrc(host string) (*Credential, error) {
	path := netrcPath()
	entries, err := parseNetrc(path)
	if err != nil {
		return nil, err
	}

	for _, entry := range entries {
		if entry.Machine == host || entry.Machine == "default" {
			return &Credential{
				Type:     CredBasic,
				Username: entry.Login,
				Password: entry.Password,
			}, nil
		}
	}
	return nil, nil
}

func netrcPath() string {
	if v := os.Getenv("NETRC"); v != "" {
		return v
	}
	home, _ := os.UserHomeDir()
	if runtime.GOOS == "windows" {
		return filepath.Join(home, "_netrc")
	}
	return filepath.Join(home, ".netrc")
}

func parseNetrc(path string) ([]NetrcEntry, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var entries []NetrcEntry
	var current *NetrcEntry

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		tokens := strings.Fields(line)
		for i := 0; i < len(tokens); i++ {
			switch tokens[i] {
			case "machine":
				if current != nil {
					entries = append(entries, *current)
				}
				current = &NetrcEntry{}
				if i+1 < len(tokens) {
					current.Machine = tokens[i+1]
					i++
				}
			case "default":
				if current != nil {
					entries = append(entries, *current)
				}
				current = &NetrcEntry{Machine: "default"}
			case "login":
				if current != nil && i+1 < len(tokens) {
					current.Login = tokens[i+1]
					i++
				}
			case "password":
				if current != nil && i+1 < len(tokens) {
					current.Password = tokens[i+1]
					i++
				}
			}
		}
	}

	if current != nil {
		entries = append(entries, *current)
	}

	return entries, scanner.Err()
}
