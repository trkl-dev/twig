// Package cmd contains all the commands for the Twig CLI
package cmd

import (
	"bytes"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/spf13/cobra"
	"github.com/trkl-dev/twig/config"
	"github.com/trkl-dev/twig/utils"
)

// ponytail: package-level perf counters, fine for a one-shot CLI.
// Atomic: runCmdIn runs from concurrent goroutines. gitTotal is summed
// across goroutines, so it can exceed wall time.
var (
	gitTotal atomic.Int64 // nanoseconds
	gitCount atomic.Int64
)

var rootCmd = &cobra.Command{
	Use:   "twig",
	Short: "Fuzzy-switch to any branch across your repos",
	Long: `Lists every branch across the repos under your project directories,
lets you pick one with fzf, checks it out in a worktree if needed,
and drops you into the directory.`,

	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.Load()
		if err != nil {
			return err
		}
		config.Cfg = cfg
		return nil
	},
	Run: func(cmd *cobra.Command, args []string) {
		runStart := time.Now()
		repos := utils.FindRepos()

		gitStart := time.Now()
		results := make([]utils.Branch, len(repos))
		var wg sync.WaitGroup
		for _, r := range repos {
			wg.Go(func() {
				results = append(results, utils.Branches(r.Path)...)
			})
		}
		wg.Wait()
		gitWall := time.Since(gitStart)

		var lines []string
		for _, ls := range results {
			lines = append(lines, ls...)
		}

		// git_total sums concurrent goroutines, so it can exceed git_wall
		slog.Debug("perf summary",
			"total_to_fzf", time.Since(runStart),
			"git_wall", gitWall,
			"git_total", time.Duration(gitTotal.Load()),
			"git_cmds", gitCount.Load(),
			"repos", len(repos),
			"branches", len(lines),
		)

		if len(lines) == 0 {
			fmt.Println("No branches found")
			return
		}

		fzfArgs := []string{
			"fzf",
			"--popup", "center,80%",
			"--prompt=branch > ",
			"--delimiter=\t",
			"--with-nth=1,3",
			"--tabstop=24",
			"--header=REPO \tBRANCH (r = remote, wt = worktree)",
		}
		fzf := exec.Command(fzfArgs[0], fzfArgs[1:]...)
		fzf.Stdin = strings.NewReader(strings.Join(lines, "\n"))
		fzf.Stderr = os.Stderr

		var buf bytes.Buffer
		fzf.Stdout = &buf

		if err := fzf.Run(); err != nil {
			os.Exit(0)
		}

		selected := strings.TrimSpace(buf.String())
		parts := strings.SplitN(selected, "\t", 4)
		if len(parts) < 4 {
			os.Exit(0)
		}
		repoName, fullBranch, repoPath := parts[0], parts[1], parts[3]
		_ = strings.TrimPrefix(fullBranch, "origin/")

		wtPath, err := utils.EnsureWorktree(repoPath, repoName, fullBranch)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}

		// stdout used to change directory
		fmt.Println(wtPath)
	},
}

// Execute runs the root command. Called by main.
func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}
