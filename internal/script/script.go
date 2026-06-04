package script

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/BurntSushi/toml"
)

type ScriptMetadata struct {
	Dependencies   []string `toml:"dependencies"`
	RequiresPython string   `toml:"requires-python"`
}

func ParseInlineMetadata(scriptPath string) (*ScriptMetadata, error) {
	f, err := os.Open(scriptPath)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	var inBlock bool
	var lines []string

	for scanner.Scan() {
		line := scanner.Text()

		if strings.TrimSpace(line) == "# /// script" {
			inBlock = true
			continue
		}

		if inBlock {
			if strings.TrimSpace(line) == "# ///" {
				break
			}
			// Strip the "# " prefix
			if strings.HasPrefix(line, "# ") {
				lines = append(lines, strings.TrimPrefix(line, "# "))
			} else if strings.HasPrefix(line, "#") {
				lines = append(lines, strings.TrimPrefix(line, "#"))
			}
		}
	}

	if !inBlock || len(lines) == 0 {
		return nil, fmt.Errorf("no inline script metadata found in %s", scriptPath)
	}

	content := strings.Join(lines, "\n")
	var meta ScriptMetadata
	if err := toml.Unmarshal([]byte(content), &meta); err != nil {
		return nil, fmt.Errorf("failed to parse script metadata: %w", err)
	}

	return &meta, nil
}

func HasInlineMetadata(scriptPath string) bool {
	meta, err := ParseInlineMetadata(scriptPath)
	return err == nil && meta != nil && len(meta.Dependencies) > 0
}
