package cli

import (
	"fmt"
	"time"

	"github.com/kartikeyyadav/fpm/internal/cache"
	"github.com/kartikeyyadav/fpm/internal/config"
	"github.com/spf13/cobra"
)

func init() {
	cacheGCCmd.RunE = runCacheGC
	cacheListUnusedCmd.RunE = runCacheListUnused
	cacheSizeCmd.RunE = runCacheSize
	cacheCleanCmd.RunE = runCacheClean
	cachePruneCmd.RunE = runCachePrune
}

func runCacheGC(cmd *cobra.Command, args []string) error {
	cfg, _ := config.LoadFromCwd()
	c := cache.New(cfg.Cache.Dir)

	olderThanStr, _ := cmd.Flags().GetString("older-than")
	var olderThan time.Duration
	if olderThanStr != "" {
		d, err := parseDuration(olderThanStr)
		if err != nil {
			return fmt.Errorf("invalid --older-than value: %w", err)
		}
		olderThan = d
	}

	result, err := c.GC(cache.GCOptions{OlderThan: olderThan})
	if err != nil {
		return err
	}

	if result.RemovedItems == 0 {
		fmt.Println("No unreferenced packages to clean up.")
		return nil
	}

	fmt.Printf("Removed %d unreferenced packages, freed %s.\n",
		result.RemovedItems, formatBytes(result.FreedBytes))
	return nil
}

func runCacheListUnused(cmd *cobra.Command, args []string) error {
	cfg, _ := config.LoadFromCwd()
	c := cache.New(cfg.Cache.Dir)
	tracker := cache.NewRefTracker(c)

	unused, err := tracker.ListUnused()
	if err != nil {
		return err
	}

	if len(unused) == 0 {
		fmt.Println("No unreferenced packages in cache.")
		return nil
	}

	fmt.Printf("Unreferenced packages (%d):\n", len(unused))
	for _, entry := range unused {
		fmt.Printf("  %s\n", entry.String())
	}

	fmt.Printf("\nRun 'fpm cache gc' to remove these.\n")
	return nil
}

func runCacheSize(cmd *cobra.Command, args []string) error {
	cfg, _ := config.LoadFromCwd()
	c := cache.New(cfg.Cache.Dir)

	size, err := c.Size()
	if err != nil {
		return err
	}

	fmt.Printf("Cache directory: %s\n\n", cfg.Cache.Dir)
	fmt.Println(size.String())
	return nil
}

func runCacheClean(cmd *cobra.Command, args []string) error {
	cfg, _ := config.LoadFromCwd()
	c := cache.New(cfg.Cache.Dir)

	if err := c.Clean(); err != nil {
		return err
	}

	fmt.Println("Cache cleared.")
	return nil
}

func runCachePrune(cmd *cobra.Command, args []string) error {
	cfg, _ := config.LoadFromCwd()
	c := cache.New(cfg.Cache.Dir)

	result, err := c.Prune()
	if err != nil {
		return err
	}

	if result.RemovedItems == 0 {
		fmt.Println("Cache is clean, nothing to prune.")
	} else {
		fmt.Printf("Pruned %d corrupted entries, freed %s.\n",
			result.RemovedItems, formatBytes(result.FreedBytes))
	}
	return nil
}

func parseDuration(s string) (time.Duration, error) {
	// Support "30d", "7d", "1h", etc.
	if len(s) == 0 {
		return 0, fmt.Errorf("empty duration")
	}

	last := s[len(s)-1]
	numStr := s[:len(s)-1]

	switch last {
	case 'd':
		var days int
		if _, err := fmt.Sscanf(numStr, "%d", &days); err != nil {
			return 0, err
		}
		return time.Duration(days) * 24 * time.Hour, nil
	case 'h':
		return time.ParseDuration(s)
	case 'm':
		return time.ParseDuration(s)
	default:
		return time.ParseDuration(s)
	}
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
