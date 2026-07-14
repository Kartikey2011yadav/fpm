package journal

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/kartikeyyadav/fpm/internal/config"
)

type Operation string

const (
	OpInstall  Operation = "install"
	OpRemove   Operation = "remove"
	OpUpgrade  Operation = "upgrade"
	OpSync     Operation = "sync"
	OpSnapshot Operation = "snapshot"
	OpRestore  Operation = "restore"
	OpInit     Operation = "init"
	OpLock     Operation = "lock"
)

type Entry struct {
	ID        string    `json:"id"`
	Timestamp time.Time `json:"timestamp"`
	Operation Operation `json:"operation"`
	Packages  []string  `json:"packages,omitempty"`
	Message   string    `json:"message,omitempty"`
	EnvPath   string    `json:"env_path,omitempty"`
}

func journalPath(envPath string) string {
	if envPath != "" {
		return filepath.Join(envPath, ".fpm-journal.jsonl")
	}
	return filepath.Join(config.DataDir(), "journal.jsonl")
}

func Record(envPath string, op Operation, packages []string, message string) {
	path := journalPath(envPath)
	os.MkdirAll(filepath.Dir(path), 0755) // journal is always user-local

	entry := Entry{
		ID:        generateID(),
		Timestamp: time.Now().UTC(),
		Operation: op,
		Packages:  packages,
		Message:   message,
		EnvPath:   envPath,
	}

	data, err := json.Marshal(entry)
	if err != nil {
		return
	}

	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return
	}
	defer f.Close()
	f.Write(append(data, '\n'))
}

func Read(envPath string, limit int) ([]Entry, error) {
	path := journalPath(envPath)
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var entries []Entry
	for _, line := range splitLines(data) {
		if len(line) == 0 {
			continue
		}
		var e Entry
		if err := json.Unmarshal(line, &e); err != nil {
			continue
		}
		entries = append(entries, e)
	}

	// Reverse (newest first)
	for i, j := 0, len(entries)-1; i < j; i, j = i+1, j-1 {
		entries[i], entries[j] = entries[j], entries[i]
	}

	if limit > 0 && len(entries) > limit {
		entries = entries[:limit]
	}

	return entries, nil
}

func generateID() string {
	return fmt.Sprintf("%x", time.Now().UnixNano()&0xFFFFFFFF)
}

func splitLines(data []byte) [][]byte {
	var lines [][]byte
	start := 0
	for i, b := range data {
		if b == '\n' {
			if i > start {
				lines = append(lines, data[start:i])
			}
			start = i + 1
		}
	}
	if start < len(data) {
		lines = append(lines, data[start:])
	}
	return lines
}
