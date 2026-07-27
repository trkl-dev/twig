package cmd

import (
	"os"
	"strings"
	"testing"
)

// Run with TEST_REPO pointing at a clone and HOME at a scratch dir.
func TestNewFlow(t *testing.T) {
	repo := os.Getenv("TEST_REPO")
	if repo == "" {
		t.Skip("TEST_REPO not set")
	}
	if base := defaultBase(repo); base != "origin/main" {
		t.Fatalf("defaultBase = %q, want origin/main", base)
	}
	wt, err := createWorktree(repo, "myrepo", "feat/thing", "origin/main")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasSuffix(wt, "/.twig/worktrees/myrepo/feat-thing") {
		t.Fatalf("unexpected worktree path %q", wt)
	}
	out, _ := runCmdIn(wt, "git", "branch", "--show-current")
	if strings.TrimSpace(out) != "feat/thing" {
		t.Fatalf("worktree on branch %q", out)
	}
	// existing branch falls through to ensureWorktree, same path back
	wt2, err := ensureWorktree(repo, "myrepo", "feat/thing")
	if err != nil || wt2 != wt {
		t.Fatalf("ensureWorktree = %q, %v; want %q", wt2, err, wt)
	}
}
