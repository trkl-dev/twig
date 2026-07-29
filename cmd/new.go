package cmd

import (
	"bufio"
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
)

// newCmd creates a branch off a repo's default branch and opens it.
var newCmd = &cobra.Command{
	Use:   "new",
	Short: "Create a new branch in a repo and open it",
	Long: `Pick a repo with fzf, name a new branch, and twig creates it off the
repo's default branch (fetched first), sets up a worktree, and opens
a tmux session — same as picking an existing branch from twig.`,
	Run: func(cmd *cobra.Command, args []string) {
		repos := findRepos()
		if len(repos) == 0 {
			fmt.Println("No repos found")
			return
		}

		var lines []string
		for _, r := range repos {
			lines = append(lines, r.name+"\t"+r.path)
		}

		fzfArgs := []string{
			"fzf",
			"--popup", "center,60%",
			"--prompt=repo > ",
			"--delimiter=\t",
			"--with-nth=1",
		}
		if os.Getenv("TMUX") == "" {
			fzfArgs = append(fzfArgs, "--border")
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

		base := defaultBase(repoPath)

		// fetch while the user types the branch name
		fetchDone := make(chan error, 1)
		if short, isRemote := strings.CutPrefix(base, "origin/"); isRemote {
			go func() {
				_, err := runCmdIn(repoPath, "git", "fetch", "origin", short)
				fetchDone <- err
			}()
		} else {
			fetchDone <- nil
		}

		fmt.Printf("new branch off %s > ", base)
		line, _ := bufio.NewReader(os.Stdin).ReadString('\n')
		branch := strings.TrimSpace(line)
		if branch == "" {
			os.Exit(0)
		}

		if out, err := runCmdIn(repoPath, "git", "check-ref-format", "--branch", branch); err != nil {
			fmt.Fprintf(os.Stderr, "invalid branch name %q: %s", branch, out)
			os.Exit(1)
		}

		if err := <-fetchDone; err != nil {
			fmt.Fprintln(os.Stderr, "warning: fetch failed, branching from local state")
		}

		var wtPath string
		var err error
		switch {
		case refExists(repoPath, "refs/heads/"+branch):
			fmt.Printf("branch %s already exists, switching to it\n", branch)
			wtPath, err = ensureWorktree(repoPath, repoName, branch)
		case refExists(repoPath, "refs/remotes/origin/"+branch):
			fmt.Printf("branch origin/%s already exists, switching to it\n", branch)
			wtPath, err = ensureWorktree(repoPath, repoName, "origin/"+branch)
		default:
			wtPath, err = createWorktree(repoPath, repoName, branch, base)
		}
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}

		if err := openTmux(sessionName(repoName, branch), wtPath); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
	},
}

func refExists(repoPath, ref string) bool {
	_, err := runCmdIn(repoPath, "git", "rev-parse", "--verify", "--quiet", ref)
	return err == nil
}

// defaultBase picks what a new branch forks from: the remote default branch
// when one is known, else the currently checked-out branch, else HEAD.
func defaultBase(repoPath string) string {
	if out, err := runCmdIn(repoPath, "git", "symbolic-ref", "--short", "refs/remotes/origin/HEAD"); err == nil {
		return strings.TrimSpace(out) // e.g. "origin/main"
	}
	for _, ref := range []string{"origin/main", "origin/master"} {
		if refExists(repoPath, "refs/remotes/"+ref) {
			return ref
		}
	}
	if out, err := runCmdIn(repoPath, "git", "branch", "--show-current"); err == nil && strings.TrimSpace(out) != "" {
		return strings.TrimSpace(out)
	}
	return "HEAD"
}

// createWorktree makes a new branch off base and checks it out in a worktree.
func createWorktree(repoPath, repoName, branch, base string) (string, error) {
	target := worktreeTarget(repoName, branch)
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		return "", err
	}
	if out, err := runCmdIn(repoPath, "git", "worktree", "add", target, "-b", branch, base); err != nil {
		return "", fmt.Errorf("git worktree add failed: %s", out)
	}
	return target, nil
}

func init() {
	rootCmd.AddCommand(newCmd)
}
