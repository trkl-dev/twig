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
	"sync"
	"sync/atomic"
	"time"

	"github.com/spf13/cobra"
)

var ProjectDirs = []string{
	"$HOME/Projects",
	"$HOME/dotfiles",
}

// MaxDepth caps how many levels below a project root the walk descends
// looking for repos.
var MaxDepth = 5

// ponytail: package-level perf counters, fine for a one-shot CLI.
// Atomic: runCmdIn runs from concurrent goroutines. gitTotal is summed
// across goroutines, so it can exceed wall time.
var (
	gitTotal atomic.Int64 // nanoseconds
	gitCount atomic.Int64
)

func runCmdIn(dir, name string, args ...string) (string, error) {
	cmd := exec.Command(name, args...)
	cmd.Dir = dir
	start := time.Now()
	out, err := cmd.CombinedOutput()
	took := time.Since(start)
	gitTotal.Add(int64(took))
	gitCount.Add(1)
	slog.Debug("ran command", "command", cmd.String(), "dir", dir, "took", took)
	return string(out), err
}

// branchLines runs `git branch -a` in a repo and formats fzf input lines.
func branchLines(repoName, repoPath string) []string {
	out, err := runCmdIn(repoPath, "git", "branch", "-a", "--format=%(refname:short)%09%(if)%(worktreepath)%(then)wt%(end)")
	if err != nil {
		slog.Warn("failed to list branches", "repo", repoName, "err", err)
		return nil
	}

	type remoteBranch struct {
		name  string
		hasWt bool
	}
	var remoteBranches []remoteBranch
	localSet := map[string]bool{}
	var lines []string

	for raw := range strings.SplitSeq(strings.TrimSpace(out), "\n") {
		b, wtFlag, _ := strings.Cut(raw, "\t")
		b = strings.TrimSpace(b)
		if b == "" || b == "origin" {
			continue
		}
		hasWt := wtFlag == "wt"
		if strings.HasPrefix(b, "origin/") {
			remoteBranches = append(remoteBranches, remoteBranch{b, hasWt})
		} else {
			display := b
			if hasWt {
				display += " (wt)"
			}
			localSet[b] = true
			slog.Debug("branch found", "branch", b)
			lines = append(lines, fmt.Sprintf("%s\t%s\t%s\t%s", repoName, b, display, repoPath))
		}
	}

	for _, rb := range remoteBranches {
		short := strings.TrimPrefix(rb.name, "origin/")
		if localSet[short] {
			continue
		}
		display := short + " (r)"
		if rb.hasWt {
			display += ", wt"
		}
		slog.Debug("branch found", "branch", rb.name)
		lines = append(lines, fmt.Sprintf("%s\t%s\t%s\t%s", repoName, rb.name, display, repoPath))
	}

	return lines
}

type repoRef struct{ name, path string }

// findRepos walks ProjectDirs and returns every git repo found.
func findRepos() []repoRef {
	var repos []repoRef
	for _, projectDir := range ProjectDirs {
		root := os.ExpandEnv(projectDir)
		fileSystem := os.DirFS(root)
		walkStart := time.Now()
		entries := 0
		fs.WalkDir(fileSystem, ".", func(path string, d fs.DirEntry, err error) error {
			entries++

			if err != nil || !d.IsDir() {
				return nil
			}
			if d.Name() == ".git" {
				return fs.SkipDir
			}
			gitPath := filepath.Join(root, path, ".git")
			gitDirInfo, err := os.Stat(gitPath)

			if err != nil {
				// not a repo; don't descend past MaxDepth
				if strings.Count(path, "/")+1 >= MaxDepth {
					return fs.SkipDir
				}
				return nil
			}

			if !gitDirInfo.IsDir() {
				slog.Debug("skipping worktree", "path", gitPath)
				return fs.SkipDir
			}

		repoPath := filepath.Join(root, path)
		repoName := d.Name()
		if path == "." {
			repoName = filepath.Base(root)
		}

		slog.Debug("repo found", "repo", repoName, "path", repoPath)
			repos = append(repos, repoRef{repoName, repoPath})

			return fs.SkipDir
		})
		slog.Debug("walk done", "root", root, "entries", entries, "took", time.Since(walkStart))
	}
	return repos
}

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "twig",
	Short: "Fuzzy-switch to any branch across your repos",
	Long: `Lists every branch across the repos under your project directories,
lets you pick one with fzf, checks it out in a worktree if needed,
and drops you into a tmux session for it.`,
	Run: func(cmd *cobra.Command, args []string) {
		runStart := time.Now()
		repos := findRepos()

		gitStart := time.Now()
		results := make([][]string, len(repos)) // indexed → deterministic order
		var wg sync.WaitGroup
		for i, r := range repos {
			wg.Go(func() {
				results[i] = branchLines(r.name, r.path)
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
		if os.Getenv("TMUX") == "" {
			fzfArgs = append(fzfArgs, "--border")
			fzfArgs = append(fzfArgs, "--height=40%")
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
		localBranch := strings.TrimPrefix(fullBranch, "origin/")

		wtPath, err := ensureWorktree(repoPath, repoName, fullBranch)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}

		if err := openTmux(sessionName(repoName, localBranch), wtPath); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
	},
}

// sessionName builds a tmux session name; tmux forbids '.' and ':' in names.
func sessionName(repoName, branch string) string {
	r := strings.NewReplacer(".", "-", ":", "-")
	return r.Replace(repoName + " » " + branch)
}

// worktreeTarget is where twig places the worktree for a repo's branch.
func worktreeTarget(repoName, branch string) string {
	return filepath.Join(os.Getenv("HOME"), ".twig", "worktrees", repoName,
		strings.ReplaceAll(branch, "/", "-"))
}

// ensureWorktree returns the path where localBranch is checked out, creating
// a worktree under ~/.twig/worktrees/<repo>/ if it isn't checked out anywhere.
func ensureWorktree(repoPath, repoName, fullBranch string) (string, error) {
	localBranch := strings.TrimPrefix(fullBranch, "origin/")

	out, err := runCmdIn(repoPath, "git", "worktree", "list", "--porcelain")
	if err != nil {
		return "", fmt.Errorf("git worktree list failed in %s: %s", repoPath, out)
	}
	var curPath string
	for line := range strings.SplitSeq(out, "\n") {
		if p, ok := strings.CutPrefix(line, "worktree "); ok {
			curPath = p
		} else if b, ok := strings.CutPrefix(line, "branch "); ok {
			if strings.TrimPrefix(b, "refs/heads/") == localBranch {
				slog.Debug("branch already checked out", "branch", localBranch, "path", curPath)
				return curPath, nil
			}
		}
	}

	target := worktreeTarget(repoName, localBranch)
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		return "", err
	}

	args := []string{"worktree", "add", target}
	if strings.HasPrefix(fullBranch, "origin/") {
		args = append(args, "-b", localBranch, fullBranch)
	} else {
		args = append(args, localBranch)
	}
	if out, err := runCmdIn(repoPath, "git", args...); err != nil {
		return "", fmt.Errorf("git worktree add failed: %s", out)
	}
	slog.Debug("worktree created", "branch", localBranch, "path", target)
	return target, nil
}

// openTmux switches (inside tmux) or attaches (outside) to the named session,
// creating it at dir first if needed.
func openTmux(name, dir string) error {
	if err := exec.Command("tmux", "has-session", "-t", "="+name).Run(); err != nil {
		if out, err := exec.Command("tmux", "new-session", "-d", "-s", name, "-c", dir).CombinedOutput(); err != nil {
			return fmt.Errorf("tmux new-session failed: %s", out)
		}
		slog.Debug("session created", "session", name, "dir", dir)
	}

	if os.Getenv("TMUX") != "" {
		if out, err := exec.Command("tmux", "switch-client", "-t", "="+name).CombinedOutput(); err != nil {
			return fmt.Errorf("tmux switch-client failed: %s", out)
		}
		return nil
	}
	attach := exec.Command("tmux", "attach-session", "-t", "="+name)
	attach.Stdin = os.Stdin
	attach.Stdout = os.Stdout
	attach.Stderr = os.Stderr
	return attach.Run()
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
