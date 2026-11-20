/*
Copyright © 2026 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/trkl-dev/twig/utils"
)

// pruneCmd represents the prune command
var pruneCmd = &cobra.Command{
	Use:   "prune",
	Short: "Prune twig worktrees",
	Long: `Remove worktrees managed by twig. Note: Worktrees with uncommitted changes cannot be pruned 
	automatically.`,

	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("prune called")
		
		worktreesHome := utils.TwigWorktreesHome()

		_ = worktreesHome


	},
}

func init() {
	rootCmd.AddCommand(pruneCmd)
}
