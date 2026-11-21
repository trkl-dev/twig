// Package cmd contains all the commands for the Twig CLI
package cmd

import (
	"bytes"
	"cmp"
	"fmt"
	"os"
	"os/exec"
	"slices"
	"strings"
	"sync"

	"github.com/spf13/cobra"
	"github.com/trkl-dev/twig/config"
	"github.com/trkl-dev/twig/utils"
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
		// This could prob be async?
		repos := utils.FindRepos()

		var wg sync.WaitGroup
		// We know that we will have at _least_ 1 branch per repo found, so may as well pre-allocate for that
		branches := make([]string, 0, len(repos))
		fmt.Printf("repos: %v\n", repos)
		c := make(chan utils.Branch, 100)

		for _, r := range repos {
			wg.Go(func() {
				utils.Branches(r.Path, c)
			})
		}

		// Wait on all the goroutines to complete and close the channel
		go func() {
			wg.Wait()
			close(c)
		}()

		for branch := range c {
			// Binary search for repoName given the repo path
			idx, found := slices.BinarySearchFunc(repos, branch.RepoPath, func(r utils.Repo, t string) int {
				return cmp.Compare(r.Path, t)
			})
			if !found {
				panic("impossible")
			}
			repo := repos[idx]
			branches = append(branches, fmt.Sprintf("%s\t%s\t%s\t%s", repo.Name, branch.Name, branch.Display(), repo.Path))
		}

		slices.Sort(branches)


		if len(branches) == 0 {
			fmt.Println("No branches found")
			return
		}

		fzfArgs := []string{
			"fzf",
			"--exact",
			"--popup", "center,80%",
			"--prompt=branch > ",
			"--delimiter=\t",
			"--with-nth=1,3",
			"--tabstop=24",
			"--header=REPO \tBRANCH (r = remote, wt = worktree)",
		}
		fzf := exec.Command(fzfArgs[0], fzfArgs[1:]...)
		fzf.Stdin = strings.NewReader(strings.Join(branches, "\n"))
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
