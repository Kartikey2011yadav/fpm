package cli

import (
	"github.com/spf13/cobra"
)

var cacheCmd = &cobra.Command{
	Use:   "cache",
	Short: "Manage the package cache",
}

var cacheCleanCmd = &cobra.Command{
	Use:   "clean",
	Short: "Remove all cached data",
}

var cachePruneCmd = &cobra.Command{
	Use:   "prune",
	Short: "Remove corrupted or invalid cache entries",
}

var cacheGCCmd = &cobra.Command{
	Use:   "gc",
	Short: "Remove unreferenced cached packages",
}

var cacheListUnusedCmd = &cobra.Command{
	Use:   "list-unused",
	Short: "Show cached packages not referenced by any environment",
}

var cacheSizeCmd = &cobra.Command{
	Use:   "size",
	Short: "Show cache size and breakdown",
}

func init() {
	cacheGCCmd.Flags().String("older-than", "", "Only GC entries unused for this duration (e.g., 30d)")
	cacheCmd.AddCommand(cacheCleanCmd)
	cacheCmd.AddCommand(cachePruneCmd)
	cacheCmd.AddCommand(cacheGCCmd)
	cacheCmd.AddCommand(cacheListUnusedCmd)
	cacheCmd.AddCommand(cacheSizeCmd)
	rootCmd.AddCommand(cacheCmd)
}
