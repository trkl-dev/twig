package cmd

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path"

	"github.com/spf13/cobra"
	"github.com/trkl-dev/twig/config"
)

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize project configuration files",
	Long: `Initialize project configuration files

	Config files will be added to $XDG_CONFIG_HOME if defined, otherwise
	$HOME/.config`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("Initializing twig...")

		configDir, err := cmd.Flags().GetString("path")
		if err != nil {
			slog.Error("Error reading 'path' flag from Cobra", "err", err)
			return
		}

		configLocation := os.ExpandEnv(configDir)

		force, err := cmd.Flags().GetBool("force")
		if err != nil {
			slog.Error("Error reading 'force' flag from Cobra", "err", err)
			return
		}

		configPath := path.Join(configLocation, "twig/config.json")
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

func init() {
	rootCmd.AddCommand(initCmd)

	initCmd.Flags().BoolP("force", "f", false, "Force initialization of project config files")

	var configLocation, exists = os.LookupEnv("XDG_CONFIG_HOME")
	if !exists {
		slog.Debug("Config env var not found")
		configLocation = os.ExpandEnv("$HOME/.config")
	}
	initCmd.Flags().StringP("path", "p", configLocation, "Write config files to custom location")
}
