package cmd

import (
	"bufio"
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/spf13/cobra"

	"github.com/trkl-dev/twig/utils"
)

// newCmd creates a branch off a repo's default branch and opens it.
var newCmd = &cobra.Command{
	Use:   "new",
	Short: "Create a new branch in a repo and open it",
	Long: `Pick a repo with fzf, name a new branch, and twig creates it off the
repo's default branch (fetched first), sets up a worktree, and changes directory 
— same as picking an existing branch from twig.`,
	Run: func(cmd *cobra.Command, args []string) {
		repos := utils.FindRepos()
		if len(repos) == 0 {
			fmt.Println("No repos found")
			return
		}

		var lines []string
		for _, r := range repos {
			lines = append(lines, r.Name+"\t"+r.Path)
		}

		fzfArgs := []string{
			"fzf",
			"--popup", "center,60%",
			"--prompt=repo > ",
			"--delimiter=\t",
			"--with-nth=1",
		}
		fzf := exec.Command(fzfArgs[0], fzfArgs[1:]...)
		fzf.Stdin = strings.NewReader(strings.Join(lines, "\n"))
		fzf.Stderr = os.Stderr
		var buf bytes.Buffer
		fzf.Stdout = &buf
		if err := fzf.Run(); err != nil {
			os.Exit(0)
		}

		repoName, repoPath, ok := strings.Cut(strings.TrimSpace(buf.String()), "\t")
		if !ok {
			os.Exit(0)
		}

		base := utils.DefaultBase(repoPath)

		// fetch while the user types the branch name
		fetchDone := make(chan error, 1)
		if short, isRemote := strings.CutPrefix(base, "origin/"); isRemote {
			go func() {
				_, err := utils.RunCmdIn(repoPath, "git", "fetch", "origin", short)
				fetchDone <- err
			}()
		} else {
			fetchDone <- nil
		}

		// Print to stderr so as not to polute the output for directory change
		fmt.Fprintf(os.Stderr, "new branch off %s > ", base)
		line, _ := bufio.NewReader(os.Stdin).ReadString('\n')
		branch := strings.TrimSpace(line)
		if branch == "" {
			os.Exit(0)
		}

		if out, err := utils.RunCmdIn(repoPath, "git", "check-ref-format", "--branch", branch); err != nil {
			fmt.Fprintf(os.Stderr, "invalid branch name %q: %s", branch, out)
			os.Exit(1)
		}

		if err := <-fetchDone; err != nil {
			fmt.Fprintln(os.Stderr, "warning: fetch failed, branching from local state")
		}

		var wtPath string
		var err error
		switch {
		case utils.RefExists(repoPath, "refs/heads/"+branch):
			fmt.Printf("branch %s already exists, switching to it\n", branch)
			wtPath, err = utils.EnsureWorktree(repoPath, repoName, branch)
		case utils.RefExists(repoPath, "refs/remotes/origin/"+branch):
			fmt.Printf("branch origin/%s already exists, switching to it\n", branch)
			wtPath, err = utils.EnsureWorktree(repoPath, repoName, "origin/"+branch)
		default:
			wtPath, err = utils.CreateWorktree(repoPath, repoName, branch, base)
		}
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		fmt.Println(wtPath)
	},
}

func init() {
	rootCmd.AddCommand(newCmd)
}
