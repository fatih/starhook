package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
)

const (
	configFile = "config.json"
	configDir  = "starhook"
)

// Config defines a physical configuration file on the host.
type Config struct {
	// Query defines the GitHub query to fetch the repositories.
	Query string `json:"query"`

	// ReposDir represents the directory to sync and manage repositories
	ReposDir string `json:"repos_dir"`

	// Token is used to communicate with the GitHub API
	Token string `json:"token"`

	path string `json:"-"`
}

// Load loads the configuration from its standad path.
func Load() (*Config, error) {
	path, err := configPath()
	if err != nil {
		return nil, err
	}

	out, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, errors.New("config file doesn't exist")
		}
		return nil, err
	}

	var cfg *Config
	err = json.Unmarshal(out, &cfg)
	if err != nil {
		return nil, err
	}

	cfg.path = path
	return cfg, nil
}

// Write writes the given config to local filesystem. If the file exists, it
// returns a ErrExists error. To overwrite the existing file, pass
// 'overwrite'.
func Write(cfg *Config, overwrite bool) error {
	path, err := configPath()
	if err != nil {
		return err
	}

	// it's safe to write if:
	// * the file doesn't exist
	// * the file exist and we allow to overwrite
	_, err = os.ReadFile(path)
	if errors.Is(err, fs.ErrNotExist) || (errors.Is(err, fs.ErrExist) && overwrite) {
		out, err := json.Marshal(cfg)
		if err != nil {
			return err
		}

		if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
			return err
		}

		err = os.WriteFile(path, out, 0600)
		if err != nil {
			fmt.Printf("err = %+v\n", err)
			return err
		}

		return nil
	}

	return err
}

func configPath() (string, error) {
	systemDir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}

	// because this is a CLI tool, it would be nice if the config lives inside
	// ~/.config instead of ~/Library/Application Support
	if runtime.GOOS == "darwin" {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}

		systemDir = filepath.Join(home, ".config")
	}

	path := filepath.Join(systemDir, configDir, configFile)
	return path, nil
}

// Path returns the absolute path of the config file's location.
func (c *Config) Path() string {
	return c.path
}
