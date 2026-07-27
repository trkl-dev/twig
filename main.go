/*
Copyright © 2026 NAME HERE <EMAIL ADDRESS>
*/
package main

import (
	"log/slog"
	"os"
	"twig/cmd"
)

func main() {
	if os.Getenv("TWIG_DEBUG") != "" {
		slog.SetLogLoggerLevel(slog.LevelDebug)
	}
	cmd.Execute()
}
