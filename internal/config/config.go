package config

import (
	"os"
	"path/filepath"
	"runtime"
)

const (
	configFile = "starhook.cue"
	configDir  = "starhook"
)

// Config defines a physical configuration file on the host.
type Config struct {
	// Query defines the GitHub query to fetch the repositories.
	Query string `json:"query"`

	// ReposDir represents the directory to sync and manage repositories
	ReposDir string `json:"repos_dir"`

	path string `json:"-"`
}

func New() (*Config, error) {
	systemDir, err := os.UserConfigDir()
	if err != nil {
		return nil, err
	}

	// because this is a CLI tool, it would be nice if the config lives inside
	// ~/.config instead of ~/Library/Application Support
	if runtime.GOOS == "darwin" {
		home, err := os.UserHomeDir()
		if err != nil {
			return nil, err
		}

		systemDir = filepath.Join(home, ".config")
	}

	return &Config{
		path: filepath.Join(systemDir, configDir, configFile),
	}, nil
}

// Path returns the absolute path of the config file's location.
func (c *Config) Path() string {
	return c.path
}
