/*
Copyright © 2026 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"bytes"
	"fmt"
	"io/fs"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
)

var ProjectDirs = []string{
	"$HOME/Projects",
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
		var lines []string

		for _, projectDir := range ProjectDirs {
			root := os.ExpandEnv(projectDir)
			fileSystem := os.DirFS(root)
			fs.WalkDir(fileSystem, ".", func(path string, d fs.DirEntry, err error) error {
				if err != nil {
					return nil
				}
				if d.Name() != ".git" || !d.IsDir() {
					return nil
				}

				repoPath := filepath.Join(root, filepath.Dir(path))
				repoName := filepath.Base(filepath.Dir(path))

				slog.Debug("repo found", "repo", repoName, "path", repoPath)

				out, err := runCmdIn(repoPath, "git", "branch", "--format=%(refname:short)")
				if err != nil {
					slog.Warn("failed to list branches", "repo", repoName, "err", err)
					return nil
				}

				for branch := range strings.SplitSeq(strings.TrimSpace(out), "\n") {
					branch = strings.TrimSpace(branch)
					if branch == "" {
						continue
					}
					lines = append(lines, fmt.Sprintf("%s\t%s", repoName, branch))
				}

				return nil
			})
		}

		if len(lines) == 0 {
			fmt.Println("No branches found")
			return
		}

		fzf := exec.Command("fzf",
			"--prompt=branch > ",
			"--delimiter=\t",
			"--with-nth=1,2",
			"--nth=1,2",
			"--tabstop=24",
			"--header=REPO | BRANCH",
			"--height=40%",
			"--border",
		)
		fzf.Stdin = strings.NewReader(strings.Join(lines, "\n"))
		fzf.Stderr = os.Stderr

		var buf bytes.Buffer
		fzf.Stdout = &buf

		if err := fzf.Run(); err != nil {
			os.Exit(0)
		}

		selected := strings.TrimSpace(buf.String())
		parts := strings.SplitN(selected, "\t", 2)
		if len(parts) == 2 {
			fmt.Printf("repo=%s  branch=%s\n", parts[0], parts[1])
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
