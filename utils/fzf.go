package utils

import (
	"bytes"
	"os"
	"os/exec"
	"strings"
)

type FzfResponse struct {
	RepoName string
	FullBranch string
	RepoPath string
} 

func Fzf(lines []string) []string {
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

	return parts
}
