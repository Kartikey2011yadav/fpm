package cache

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"math/rand"
	"os"
	"path/filepath"
	"strconv"

	"github.com/kartikeyyadav/fpm/internal/config"
)

type Cache struct {
	Root string
}

type CASKey struct {
	Algorithm string
	Digest    string
}

func (k CASKey) String() string {
	return k.Algorithm + ":" + k.Digest
}

func (k CASKey) Prefix() string {
	if len(k.Digest) >= 2 {
		return k.Digest[:2]
	}
	return k.Digest
}

func (k CASKey) IsZero() bool {
	return k.Digest == ""
}

func New(root string) *Cache {
	if root == "" {
		root = config.CacheDir()
	}
	return &Cache{Root: root}
}

func (c *Cache) Init() error {
	perm := os.FileMode(0755)
	if config.IsMultiUserMode() {
		perm = 0775 | os.ModeSetgid
	}
	dirs := []string{
		c.CASDir(),
		c.WheelsDir(),
		c.HTTPDir(),
		c.RefsDir(),
		c.TmpDir(),
		c.InterpretersDir(),
		c.OSVDir(),
		c.GitDir(),
		c.EnvironmentsDir(),
	}
	for _, d := range dirs {
		if err := os.MkdirAll(d, perm); err != nil {
			return fmt.Errorf("failed to create cache dir %s: %w", d, err)
		}
	}
	return nil
}

func (c *Cache) CASDir() string          { return filepath.Join(c.Root, "cas", "sha256") }
func (c *Cache) WheelsDir() string       { return filepath.Join(c.Root, "wheels") }
func (c *Cache) HTTPDir() string         { return filepath.Join(c.Root, "http") }
func (c *Cache) RefsDir() string         { return filepath.Join(c.Root, "refs") }
func (c *Cache) TmpDir() string          { return filepath.Join(c.Root, "tmp") }
func (c *Cache) InterpretersDir() string { return filepath.Join(c.Root, "interpreters") }
func (c *Cache) OSVDir() string          { return filepath.Join(c.Root, "osv") }
func (c *Cache) GitDir() string          { return filepath.Join(c.Root, "git") }
func (c *Cache) EnvironmentsDir() string { return filepath.Join(c.Root, "environments") }

func (c *Cache) CASPath(key CASKey) string {
	return filepath.Join(c.CASDir(), key.Prefix(), key.Digest)
}

func (c *Cache) Has(key CASKey) bool {
	_, err := os.Stat(c.CASPath(key))
	return err == nil
}

func (c *Cache) Store(wheelPath string) (CASKey, error) {
	hash, err := hashFile(wheelPath)
	if err != nil {
		return CASKey{}, fmt.Errorf("failed to hash wheel: %w", err)
	}

	key := CASKey{Algorithm: "sha256", Digest: hash}

	// Check if already stored
	casPath := c.CASPath(key)
	if _, err := os.Stat(casPath); err == nil {
		return key, nil // already exists
	}

	// Extract to temp with unique path (PID + random), then atomic rename
	suffix := strconv.Itoa(os.Getpid()) + "-" + strconv.FormatInt(rand.Int63(), 36)
	tmpPath := filepath.Join(c.TmpDir(), "extract-"+hash+"-"+suffix)
	os.RemoveAll(tmpPath)
	if err := os.MkdirAll(tmpPath, 0755); err != nil {
		return CASKey{}, err
	}

	if err := extractWheelToDir(wheelPath, tmpPath); err != nil {
		os.RemoveAll(tmpPath)
		return CASKey{}, fmt.Errorf("failed to extract wheel: %w", err)
	}

	// Atomic move to CAS location
	if err := os.MkdirAll(filepath.Dir(casPath), 0755); err != nil {
		os.RemoveAll(tmpPath)
		return CASKey{}, err
	}

	if err := os.Rename(tmpPath, casPath); err != nil {
		os.RemoveAll(tmpPath)
		return CASKey{}, fmt.Errorf("failed to move to CAS: %w", err)
	}

	return key, nil
}

func (c *Cache) Retrieve(key CASKey) (string, error) {
	path := c.CASPath(key)
	if _, err := os.Stat(path); err != nil {
		return "", fmt.Errorf("CAS entry not found: %s", key)
	}
	return path, nil
}

func (c *Cache) Remove(key CASKey) error {
	return os.RemoveAll(c.CASPath(key))
}

func (c *Cache) Clean() error {
	entries := []string{c.CASDir(), c.WheelsDir(), c.HTTPDir(), c.RefsDir(), c.TmpDir()}
	for _, e := range entries {
		os.RemoveAll(e)
	}
	return c.Init()
}

// HashFile computes the SHA256 hex digest of a file.
func HashFile(path string) (string, error) {
	return hashFile(path)
}

func hashFile(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()

	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

func extractWheelToDir(wheelPath, destDir string) error {
	// Delegate to wheel package's extraction
	// For now, use archive/zip directly
	return extractZip(wheelPath, destDir)
}
