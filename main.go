/*
Copyright © 2026 NAME HERE <EMAIL ADDRESS>
*/
package main

import (
	"log/slog"
	"twig/cmd"
)

func main() {
	slog.SetLogLoggerLevel(slog.LevelInfo)
	cmd.Execute()
}
