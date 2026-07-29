package main

import (
	"log/slog"
	"os"
	"github.com/trkl-dev/twig/cmd"
)

func main() {
	if os.Getenv("TWIG_DEBUG") != "" {
		slog.SetLogLoggerLevel(slog.LevelDebug)
	}
	cmd.Execute()
}
