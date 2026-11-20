package cmd

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path"

	"github.com/spf13/cobra"
	"github.com/trkl-dev/twig/config"
	"github.com/trkl-dev/twig/utils"

	_ "embed"
)

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize project configuration files",
	Long: `Initialize project configuration files

	Config files will be added to $XDG_CONFIG_HOME if defined, otherwise
	$HOME/.config`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("Initializing twig...")

		configLocation := utils.TwigConfigHome()

		force, err := cmd.Flags().GetBool("force")
		if err != nil {
			slog.Error("Error reading 'force' flag from Cobra", "err", err)
			return
		}

		configParentPath := path.Join(configLocation, "twig")

		err = os.MkdirAll(configParentPath, os.ModePerm)
		if err != nil {
			slog.Error("Error ensuring config directory path is present", "err", err)
			return
		}

		configPath := path.Join(configParentPath, "config.json")
		slog.Debug("config path", "path", configPath)
		_, err = os.Stat(configPath)
		if os.IsNotExist(err) || force {
			configData, err := json.MarshalIndent(config.DefaultConfig(), "", "    ")
			if err != nil {
				slog.Error("Error marshalling JSON", "err", err)
				return
			}
			err = os.WriteFile(configPath, configData, 0644)
			if err != nil {
				slog.Error("Error writing config JSON", "err", err)
				return
			}
			fmt.Println("Twig initialization complete.")
			return
		}

		fmt.Println("Twig already initialized.")

	},
}

//go:embed shells/bash.sh
var bashFile string

//go:embed shells/twig.fish
var fishFile string

var initBash = &cobra.Command{
	Use:   "bash",
	Short: "Output bash script for shell loading",

	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println(string(bashFile))
	},
}

var initZsh = &cobra.Command{
	Use:   "zsh",
	Short: "Output zsh script for shell loading",

	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println(string(bashFile))
	},
}

var initFish = &cobra.Command{
	Use:   "fish",
	Short: "Output fish script for shell loading",

	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println(string(fishFile))
	},
}

func init() {
	rootCmd.AddCommand(initCmd)

	initCmd.Flags().BoolP("force", "f", false, "Force initialization of project config files and override any existing config")

	initCmd.AddCommand(initBash)
	initCmd.AddCommand(initZsh)
	initCmd.AddCommand(initFish)
}
