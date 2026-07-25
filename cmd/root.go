/*
Copyright © 2026 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"fmt"
	"io/fs"
	"log"
	"log/slog"
	"os"
	"os/exec"
	"strings"

	"github.com/spf13/cobra"
)

var ProjectDirs = []string{
	// "~/Projects",
	"$HOME/Projects",
	// "/home/nick/Projects",
}

func runCmdIn(dir, name string, args ...string) (string, error) {
	cmd := exec.Command(name, args...)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	slog.Debug("running command", "command", cmd.String(), "dir", dir)
	return string(out), err
}

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "twig",
	Short: "A brief description of your application",
	Long: `A longer description that spans multiple lines and likely contains
examples and usage of using your application. For example:

Cobra is a CLI library for Go that empowers applications.
This application is a tool to generate the needed files
to quickly create a Cobra application.`,
	Run: func(cmd *cobra.Command, args []string) {
		// Search for all repos
		for _, projectDir := range ProjectDirs {
			slog.Debug("walking tree", "dir", projectDir)
			root := os.ExpandEnv(projectDir)
			fileSystem := os.DirFS(root)
			err := fs.WalkDir(fileSystem, ".", func(path string, d fs.DirEntry, err error) error {
				if err != nil {
					log.Fatal(err)
				}
				if d.Name() != ".git" {
					return nil
				}
				if !d.IsDir() {
					return nil
				}

				splitPath := strings.Split(path, "/")
				parentDir := splitPath[len(splitPath)-2]
				parentPathRelative := strings.Replace(path, "/.git", "", 1)
				parentPathAbsolute := fmt.Sprintf("%s/%s", root, parentPathRelative)


				slog.Info("git repo found", "path", path, "parent", parentDir)

				gitBranchesRaw, err := runCmdIn(parentPathAbsolute, "git", "branch", "-a")
				if err != nil {
					return err
				}

				gitBranches := strings.Split(gitBranchesRaw, "\n")

				slog.Info("git branches found")
				for _, branch := range gitBranches {
					slog.Info(">", "branch", strings.TrimSpace(branch))
				}


				return nil
			})
			if err != nil {
				log.Fatal(err)
			}
		}
	},
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func init() {
	// Here you will define your flags and configuration settings.
	// Cobra supports persistent flags, which, if defined here,
	// will be global for your application.

	// rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.twig.yaml)")

	// Cobra also supports local flags, which will only run
	// when this action is called directly.
	rootCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}
