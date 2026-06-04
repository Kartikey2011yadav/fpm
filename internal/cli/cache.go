package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

var cacheCmd = &cobra.Command{
	Use:   "cache",
	Short: "Manage the package cache",
}

var cacheCleanCmd = &cobra.Command{
	Use:   "clean",
	Short: "Remove all cached data",
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Println("fpm cache clean: not yet implemented")
		return nil
	},
}

var cachePruneCmd = &cobra.Command{
	Use:   "prune",
	Short: "Remove corrupted or invalid cache entries",
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Println("fpm cache prune: not yet implemented")
		return nil
	},
}

var cacheGCCmd = &cobra.Command{
	Use:   "gc",
	Short: "Remove unreferenced cached packages",
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Println("fpm cache gc: not yet implemented")
		return nil
	},
}

var cacheListUnusedCmd = &cobra.Command{
	Use:   "list-unused",
	Short: "Show cached packages not referenced by any environment",
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Println("fpm cache list-unused: not yet implemented")
		return nil
	},
}

var cacheSizeCmd = &cobra.Command{
	Use:   "size",
	Short: "Show cache size and breakdown",
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Println("fpm cache size: not yet implemented")
		return nil
	},
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
