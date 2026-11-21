package utils

import (
	"cmp"
	"fmt"
	"io/fs"
	"log/slog"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"github.com/trkl-dev/twig/config"
)

func RefExists(repoPath, ref string) bool {
	_, err := RunCmdIn(repoPath, "git", "rev-parse", "--verify", "--quiet", ref)
	return err == nil
}

// defaultBase picks what a new branch forks from: the remote default branch
// when one is known, else the currently checked-out branch, else HEAD.
func DefaultBase(repoPath string) string {
	if out, err := RunCmdIn(repoPath, "git", "symbolic-ref", "--short", "refs/remotes/origin/HEAD"); err == nil {
		return strings.TrimSpace(out) // e.g. "origin/main"
	}
	for _, ref := range []string{"origin/main", "origin/master"} {
		if RefExists(repoPath, "refs/remotes/"+ref) {
			return ref
		}
	}
	if out, err := RunCmdIn(repoPath, "git", "branch", "--show-current"); err == nil && strings.TrimSpace(out) != "" {
		return strings.TrimSpace(out)
	}
	return "HEAD"
}

// createWorktree makes a new branch off base and checks it out in a worktree.
func CreateWorktree(repoPath, repoName, branch, base string) (string, error) {
	target := TwigWorktreeLocation(repoName, branch)
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		return "", err
	}
	if out, err := RunCmdIn(repoPath, "git", "worktree", "add", target, "-b", branch, base); err != nil {
		return "", fmt.Errorf("git worktree add failed: %s", out)
	}
	return target, nil
}

func TwigConfigHome() string {
	// Deliberately not using os.UserConfigDir because it doesn't feel right on mac.
	// Still allow override through XDG_CONFIG_HOME
	var configLocation, exists = os.LookupEnv("XDG_CONFIG_HOME")
	if !exists {
		slog.Debug("Config env var not found")
		configLocation = os.ExpandEnv("$HOME/.config")
	}
	return configLocation
}

// worktreeTarget is where twig places the worktree for a repo's branch.
func TwigWorktreesHome() string {
	return filepath.Join(os.Getenv("HOME"), ".twig", "worktrees")
}

// worktreeTarget is where twig places the worktree for a repo's branch.
func TwigWorktreeLocation(repoName, branch string) string {
	return filepath.Join(os.Getenv("HOME"), ".twig", "worktrees", repoName,
		strings.ReplaceAll(branch, "/", "-"))
}

type Branch struct {
	Name     string
	HasWT    bool // QUES: Could you argue this should also be an array to look up into?
	RepoPath string
	IsRemote bool
}

// TODO: Sense checks:
// - Branch can't be remote and have a worktree

func (b *Branch) Display() string {
	if b.HasWT && b.IsRemote {
		panic("impossible")
	}
	var tag string
	if b.HasWT {
		tag = "(wt)"
	}
	if b.IsRemote {
		tag = "(r)"
	}
	return fmt.Sprintf("%s %s", b.Name, tag)
}

// branchLines runs `git branch -a` in a repo and formats fzf input lines.
func Branches(repoPath string, c chan Branch) {
	slog.Debug("Retrieving branches", "repoPath", repoPath)
	out, err := RunCmdIn(repoPath, "git", "branch", "-a", "--format=%(refname:short)%09%(if)%(worktreepath)%(then)wt%(end)")
	if err != nil {
		slog.Warn("failed to list branches", "repo", repoPath, "err", err)
		close(c)
		return
	}

	for raw := range strings.SplitSeq(strings.TrimSpace(out), "\n") {
		branchName, wtFlag, _ := strings.Cut(raw, "\t")
		branchName = strings.TrimSpace(branchName)
		if branchName == "" || branchName == "origin" {
			continue
		}
		hasWt := wtFlag == "wt"
		isRemote := strings.HasPrefix(branchName, "origin/")
		if hasWt && isRemote {
			panic("Branch has both worktree and marked as remote")
		}

		c <- Branch{
			Name:     branchName,
			HasWT:    hasWt,
			RepoPath: repoPath,
			IsRemote: isRemote,
		}
	}
}

type Repo struct {
	Path string // Uniquely identifies
	Name string
}

// FIXME: I think this comes from the config now?
// MaxDepth caps how many levels below a project root the walk descends
// looking for repos.
var MaxDepth = 5

// findRepos walks ProjectDirs and returns every git repo found.
func FindRepos() []Repo {
	var repos []Repo
	for _, projectDir := range config.Cfg.ProjectDirs {
		root := os.ExpandEnv(projectDir)
		fileSystem := os.DirFS(root)
		walkStart := time.Now()
		entries := 0
		err := fs.WalkDir(fileSystem, ".", func(path string, d fs.DirEntry, err error) error {
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
			repos = append(repos, Repo{
				Path: repoPath,
				Name: repoName,
			})

			return fs.SkipDir
		})
		slog.Debug("walk done", "root", root, "entries", entries, "took", time.Since(walkStart))
		if err != nil {
			slog.Error("err", "err", err) // FIXME: ew
			os.Exit(0)
		}
	}

	// Return the slice sorted so we can search quicker
	slices.SortFunc(repos, func(a, b Repo) int {
		return cmp.Compare(a.Path, b.Path)
	})

	return repos
}

// ensureWorktree returns the path where localBranch is checked out, creating
// a worktree under ~/.twig/worktrees/<repo>/ if it isn't checked out anywhere.
func EnsureWorktree(repoPath, repoName, fullBranch string) (string, error) {
	localBranch := strings.TrimPrefix(fullBranch, "origin/")

	out, err := RunCmdIn(repoPath, "git", "worktree", "list", "--porcelain")
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

	target := TwigWorktreeLocation(repoName, localBranch)
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		return "", err
	}

	args := []string{"worktree", "add", target}
	if strings.HasPrefix(fullBranch, "origin/") {
		args = append(args, "-b", localBranch, fullBranch)
	} else {
		args = append(args, localBranch)
	}
	if out, err := RunCmdIn(repoPath, "git", args...); err != nil {
		return "", fmt.Errorf("git worktree add failed: %s", out)
	}
	slog.Debug("worktree created", "branch", localBranch, "path", target)
	return target, nil
}
