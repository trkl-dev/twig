package config

import (
	"encoding/json"
	"io"
	"log/slog"
	"os"
	"path"
)

type Config struct {
	ProjectDirs  []string `json:"project_dirs"`
	MaxDepth     int      `json:"max_depth"`
	WorktreeRoot string   `json:"worktree_root"`
}

var Cfg Config


// TODO: Revisit this. I'm not sure the default ProjectDirs makes sense
func DefaultConfig() Config {
	return Config{
		ProjectDirs: []string{
			os.ExpandEnv("$HOME/Projects"),
		},
		MaxDepth:     5,
		WorktreeRoot: os.ExpandEnv("$HOME/.twig/worktrees"),
	}
}

func Load() (Config, error) {
	var configLocation, exists = os.LookupEnv("XDG_CONFIG_HOME")
	if !exists {
		slog.Warn("Config env var not found")
		configLocation = os.ExpandEnv("$HOME/.config")
	}

	configPath := path.Join(configLocation, "twig/config.json")
	configJson, err := os.Open(configPath)
	if err != nil {
		slog.Warn("Config file not found", "path", configPath, "err", err)
		return DefaultConfig(), nil 
	}
	defer configJson.Close()

	jsonBytes, err := io.ReadAll(configJson)
	if err != nil {
		slog.Warn("Config could not be read", "err", err)
		return DefaultConfig(), err
	}

	config := Config{}
	err = json.Unmarshal(jsonBytes, &config)
	if err != nil {
		slog.Warn("Error unmarshalling json file", "err", err)
		return DefaultConfig(), err
	}

	return config, nil
}
