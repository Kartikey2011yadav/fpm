package cache

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/kartikeyyadav/fpm/internal/config"
	"github.com/kartikeyyadav/fpm/internal/fs"
)

type RefTracker struct {
	cache *Cache
}

type EnvRef struct {
	CASKeys   []string  `json:"cas_keys"`
	UpdatedAt time.Time `json:"updated_at"`
}

type CASRef struct {
	Environments []string  `json:"environments"`
	PackageName  string    `json:"package_name"`
	Version      string    `json:"version"`
	UpdatedAt    time.Time `json:"updated_at"`
}

func NewRefTracker(c *Cache) *RefTracker {
	return &RefTracker{cache: c}
}

func (r *RefTracker) AddReference(envPath string, key CASKey, pkgName, version string) error {
	envHash := hashString(envPath)

	// Update by-env index
	if err := r.addToEnvIndex(envHash, key); err != nil {
		return err
	}

	// Update by-cas index
	return r.addToCASIndex(key, envPath, pkgName, version)
}

func (r *RefTracker) RemoveReference(envPath string, key CASKey) error {
	envHash := hashString(envPath)

	// Update by-env index
	if err := r.removeFromEnvIndex(envHash, key); err != nil {
		return err
	}

	// Update by-cas index
	return r.removeFromCASIndex(key, envPath)
}

func (r *RefTracker) RemoveEnvironment(envPath string) error {
	envHash := hashString(envPath)
	envRefPath := r.envRefPath(envHash)

	ref, err := r.readEnvRef(envHash)
	if err != nil {
		return nil // no refs to clean up
	}

	// Remove from all CAS indexes
	for _, keyStr := range ref.CASKeys {
		key := parseCASKey(keyStr)
		r.removeFromCASIndex(key, envPath)
	}

	return os.Remove(envRefPath)
}

func (r *RefTracker) ReferenceCount(key CASKey) (int, error) {
	ref, err := r.readCASRef(key)
	if err != nil {
		return 0, nil
	}
	return len(ref.Environments), nil
}

func (r *RefTracker) ListUnused() ([]UnusedEntry, error) {
	casDir := r.cache.CASDir()
	var unused []UnusedEntry

	prefixDirs, err := os.ReadDir(casDir)
	if err != nil {
		return nil, nil
	}

	for _, prefix := range prefixDirs {
		if !prefix.IsDir() {
			continue
		}
		subDir := filepath.Join(casDir, prefix.Name())
		entries, err := os.ReadDir(subDir)
		if err != nil {
			continue
		}

		for _, entry := range entries {
			if !entry.IsDir() {
				continue
			}
			key := CASKey{Algorithm: "sha256", Digest: prefix.Name() + entry.Name()[0:]}
			// Reconstruct the full digest from prefix + rest
			key.Digest = entry.Name()

			casRef, err := r.readCASRef(key)
			if err != nil || len(casRef.Environments) == 0 {
				info, _ := entry.Info()
				var size int64
				if info != nil {
					dirPath := filepath.Join(subDir, entry.Name())
					size, _ = dirSize(dirPath)
				}
				unused = append(unused, UnusedEntry{
					Key:         key,
					PackageName: casRef.PackageName,
					Version:     casRef.Version,
					Size:        size,
				})
			}
		}
	}

	return unused, nil
}

type UnusedEntry struct {
	Key         CASKey
	PackageName string
	Version     string
	Size        int64
}

func (u UnusedEntry) String() string {
	name := u.PackageName
	if name == "" {
		name = u.Key.Digest[:12]
	}
	if u.Version != "" {
		name += " " + u.Version
	}
	return fmt.Sprintf("%s (%s)", name, formatBytes(u.Size))
}

// Internal helpers

func (r *RefTracker) envRefsDir() string {
	return filepath.Join(r.cache.RefsDir(), "by-env")
}

func (r *RefTracker) casRefsDir() string {
	return filepath.Join(r.cache.RefsDir(), "by-cas")
}

func (r *RefTracker) envRefPath(envHash string) string {
	return filepath.Join(r.envRefsDir(), envHash+".json")
}

func (r *RefTracker) casRefPath(key CASKey) string {
	return filepath.Join(r.casRefsDir(), key.Algorithm+"-"+key.Digest+".json")
}

func (r *RefTracker) readEnvRef(envHash string) (*EnvRef, error) {
	data, err := os.ReadFile(r.envRefPath(envHash))
	if err != nil {
		return nil, err
	}
	var ref EnvRef
	return &ref, json.Unmarshal(data, &ref)
}

func (r *RefTracker) writeEnvRef(envHash string, ref *EnvRef) error {
	os.MkdirAll(r.envRefsDir(), 0755)
	path := r.envRefPath(envHash)
	ref.UpdatedAt = time.Now()
	data, err := json.MarshalIndent(ref, "", "  ")
	if err != nil {
		return err
	}
	filePerm := os.FileMode(0644)
	if config.IsMultiUserMode() {
		filePerm = 0664
	}
	return os.WriteFile(path, data, filePerm)
}

func (r *RefTracker) readCASRef(key CASKey) (*CASRef, error) {
	data, err := os.ReadFile(r.casRefPath(key))
	if err != nil {
		return &CASRef{}, err
	}
	var ref CASRef
	return &ref, json.Unmarshal(data, &ref)
}

func (r *RefTracker) writeCASRef(key CASKey, ref *CASRef) error {
	os.MkdirAll(r.casRefsDir(), 0755)
	path := r.casRefPath(key)
	ref.UpdatedAt = time.Now()
	data, err := json.MarshalIndent(ref, "", "  ")
	if err != nil {
		return err
	}
	filePerm := os.FileMode(0644)
	if config.IsMultiUserMode() {
		filePerm = 0664
	}
	return os.WriteFile(path, data, filePerm)
}

func (r *RefTracker) addToEnvIndex(envHash string, key CASKey) error {
	os.MkdirAll(r.envRefsDir(), 0755)
	path := r.envRefPath(envHash)
	lock, err := fs.LockFile(path)
	if err != nil {
		return fmt.Errorf("failed to acquire env ref lock: %w", err)
	}
	defer fs.UnlockFile(lock)

	ref, _ := r.readEnvRef(envHash)
	if ref == nil {
		ref = &EnvRef{}
	}

	keyStr := key.String()
	for _, k := range ref.CASKeys {
		if k == keyStr {
			return nil
		}
	}
	ref.CASKeys = append(ref.CASKeys, keyStr)
	return r.writeEnvRef(envHash, ref)
}

func (r *RefTracker) removeFromEnvIndex(envHash string, key CASKey) error {
	os.MkdirAll(r.envRefsDir(), 0755)
	path := r.envRefPath(envHash)
	lock, err := fs.LockFile(path)
	if err != nil {
		return nil
	}
	defer fs.UnlockFile(lock)

	ref, err := r.readEnvRef(envHash)
	if err != nil {
		return nil
	}

	keyStr := key.String()
	filtered := ref.CASKeys[:0]
	for _, k := range ref.CASKeys {
		if k != keyStr {
			filtered = append(filtered, k)
		}
	}
	ref.CASKeys = filtered
	return r.writeEnvRef(envHash, ref)
}

func (r *RefTracker) addToCASIndex(key CASKey, envPath, pkgName, version string) error {
	os.MkdirAll(r.casRefsDir(), 0755)
	path := r.casRefPath(key)
	lock, err := fs.LockFile(path)
	if err != nil {
		return fmt.Errorf("failed to acquire CAS ref lock: %w", err)
	}
	defer fs.UnlockFile(lock)

	ref, _ := r.readCASRef(key)
	if ref == nil {
		ref = &CASRef{}
	}

	for _, e := range ref.Environments {
		if e == envPath {
			return nil
		}
	}
	ref.Environments = append(ref.Environments, envPath)
	ref.PackageName = pkgName
	ref.Version = version
	return r.writeCASRef(key, ref)
}

func (r *RefTracker) removeFromCASIndex(key CASKey, envPath string) error {
	os.MkdirAll(r.casRefsDir(), 0755)
	path := r.casRefPath(key)
	lock, err := fs.LockFile(path)
	if err != nil {
		return nil
	}
	defer fs.UnlockFile(lock)

	ref, err := r.readCASRef(key)
	if err != nil {
		return nil
	}

	filtered := ref.Environments[:0]
	for _, e := range ref.Environments {
		if e != envPath {
			filtered = append(filtered, e)
		}
	}
	ref.Environments = filtered
	return r.writeCASRef(key, ref)
}

// FindCASKeyByPackage searches the CAS refs directory for a matching package name and version,
// returning the CAS digest if found.
func (r *RefTracker) FindCASKeyByPackage(pkgName, version string) string {
	refsDir := r.casRefsDir()
	entries, err := os.ReadDir(refsDir)
	if err != nil {
		return ""
	}
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		data, err := os.ReadFile(filepath.Join(refsDir, entry.Name()))
		if err != nil {
			continue
		}
		var ref CASRef
		if json.Unmarshal(data, &ref) != nil {
			continue
		}
		if ref.PackageName == pkgName && ref.Version == version {
			// Extract the digest from filename: "sha256-<digest>.json"
			name := strings.TrimSuffix(entry.Name(), ".json")
			parts := strings.SplitN(name, "-", 2)
			if len(parts) == 2 {
				return parts[1]
			}
		}
	}
	return ""
}

func hashString(s string) string {
	h := sha256.Sum256([]byte(s))
	return hex.EncodeToString(h[:])[:16]
}

func parseCASKey(s string) CASKey {
	parts := strings.SplitN(s, ":", 2)
	if len(parts) != 2 {
		return CASKey{Algorithm: "sha256", Digest: s}
	}
	return CASKey{Algorithm: parts[0], Digest: parts[1]}
}

func dirSize(path string) (int64, error) {
	var size int64
	filepath.Walk(path, func(_ string, info os.FileInfo, _ error) error {
		if info != nil && !info.IsDir() {
			size += info.Size()
		}
		return nil
	})
	return size, nil
}

func formatBytes(b int64) string {
	const unit = 1024
	if b < unit {
		return fmt.Sprintf("%d B", b)
	}
	div, exp := int64(unit), 0
	for n := b / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(b)/float64(div), "KMGTPE"[exp])
}
