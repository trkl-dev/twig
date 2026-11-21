package utils

import (
	"log/slog"
	"os/exec"
	"time"
)


func RunCmdIn(dir, name string, args ...string) (string, error) {
	cmd := exec.Command(name, args...)
	cmd.Dir = dir
	start := time.Now()
	out, err := cmd.CombinedOutput()
	took := time.Since(start)
	slog.Debug("ran command", "command", cmd.String(), "dir", dir, "took", took)
	return string(out), err
}

