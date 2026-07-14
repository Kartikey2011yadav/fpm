package cache

import (
	"fmt"
	"os"
	"time"
)

type GCOptions struct {
	OlderThan time.Duration
	DryRun    bool
}

type GCResult struct {
	RemovedItems int
	SkippedItems int
	FreedBytes   int64
	Entries      []UnusedEntry
}

func (c *Cache) GC(opts GCOptions) (*GCResult, error) {
	tracker := NewRefTracker(c)
	unused, err := tracker.ListUnused()
	if err != nil {
		return nil, fmt.Errorf("failed to list unused entries: %w", err)
	}

	result := &GCResult{}

	for _, entry := range unused {
		// Apply age filter if specified
		if opts.OlderThan > 0 {
			casPath := c.CASPath(entry.Key)
			info, err := os.Stat(casPath)
			if err != nil {
				continue
			}
			age := time.Since(info.ModTime())
			if age < opts.OlderThan {
				continue
			}
		}

		result.Entries = append(result.Entries, entry)
		result.FreedBytes += entry.Size

		if !opts.DryRun {
			if err := c.Remove(entry.Key); err != nil {
				if os.IsPermission(err) {
					result.SkippedItems++
				}
				continue
			}
			refPath := NewRefTracker(c).casRefPath(entry.Key)
			os.Remove(refPath)
			result.RemovedItems++
		} else {
			result.RemovedItems++
		}
	}

	return result, nil
}

func (c *Cache) Prune() (*GCResult, error) {
	// Prune only validates integrity — removes corrupted entries
	result := &GCResult{}

	// Walk CAS and verify each entry has valid structure
	casDir := c.CASDir()
	prefixDirs, err := os.ReadDir(casDir)
	if err != nil {
		return result, nil
	}

	for _, prefix := range prefixDirs {
		if !prefix.IsDir() {
			continue
		}
		subDir := fmt.Sprintf("%s/%s", casDir, prefix.Name())
		entries, err := os.ReadDir(subDir)
		if err != nil {
			continue
		}

		for _, entry := range entries {
			if !entry.IsDir() {
				// Stray file in CAS, remove it
				path := fmt.Sprintf("%s/%s", subDir, entry.Name())
				info, _ := os.Stat(path)
				if info != nil {
					result.FreedBytes += info.Size()
				}
				os.Remove(path)
				result.RemovedItems++
			}
		}
	}

	// Clean tmp directory
	tmpDir := c.TmpDir()
	entries, _ := os.ReadDir(tmpDir)
	for _, entry := range entries {
		path := fmt.Sprintf("%s/%s", tmpDir, entry.Name())
		size, _ := dirSize(path)
		result.FreedBytes += size
		os.RemoveAll(path)
		result.RemovedItems++
	}

	return result, nil
}

type CacheSize struct {
	Total  int64
	CAS    int64
	Wheels int64
	HTTP   int64
	Refs   int64
	Tmp    int64
}

func (s CacheSize) String() string {
	return fmt.Sprintf("Total: %s\n  CAS: %s\n  Wheels: %s\n  HTTP: %s\n  Refs: %s\n  Tmp: %s",
		formatBytes(s.Total),
		formatBytes(s.CAS),
		formatBytes(s.Wheels),
		formatBytes(s.HTTP),
		formatBytes(s.Refs),
		formatBytes(s.Tmp),
	)
}

func (c *Cache) Size() (*CacheSize, error) {
	size := &CacheSize{}

	size.CAS, _ = dirSize(c.CASDir())
	size.Wheels, _ = dirSize(c.WheelsDir())
	size.HTTP, _ = dirSize(c.HTTPDir())
	size.Refs, _ = dirSize(c.RefsDir())
	size.Tmp, _ = dirSize(c.TmpDir())
	size.Total = size.CAS + size.Wheels + size.HTTP + size.Refs + size.Tmp

	return size, nil
}
