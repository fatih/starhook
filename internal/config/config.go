package config

import (
	"encoding/json"
	"errors"
	"fmt"
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
	// Selected defines the name of the selected config.
	Selected string `json:"selected"`

	RepoSets []*RepoSet `json:"repo_sets"`

	path string `json:"-"`
}

// RepoSet defines a single configuration that represents a set of repositories
// and the query used to fetch the repositories
type RepoSet struct {
	// Name is a logical name to represent this config.
	Name string `json:"name"`

	// Query defines the GitHub query to fetch the repositories.
	Query string `json:"query"`

	// ReposDir represents the directory to sync and manage repositories
	ReposDir string `json:"repos_dir"`

	// Token is used to communicate with the GitHub API
	Token string `json:"token"`
}

// Load loads the configuration from its standard path.
func Load() (*Config, error) {
	path, err := configPath()
	if err != nil {
		return nil, err
	}

	out, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("config file doesn't exist: %w", err)
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

// Create creates a new, empty configuration file. The user should populate the
// config afterwards.
func Create() (*Config, error) {
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

// Path returns the absolute path of the config file's location.
func (c *Config) Path() string {
	return c.path
}

// SelectedRepoSet returns the selected reposet, if available
func (c *Config) SelectedRepoSet() (*RepoSet, error) {
	var rs *RepoSet
	for _, r := range c.RepoSets {
		if r.Name == c.Selected {
			rs = r
		}
	}

	if rs == nil {
		return nil, fmt.Errorf("couldn't find repo set configuration for %q", c.Selected)
	}

	return rs, nil
}

func (c *Config) AddRepoSet(rs *RepoSet, force bool) error {
	hasRepoSet := false

	for i, set := range c.RepoSets {
		if set.Name == rs.Name {
			hasRepoSet = true

			if force {
				c.RepoSets[i] = rs // overwrite
			} else {
				return fmt.Errorf("repo set with name %q already exists", rs.Name)
			}
		}
	}

	// if there is no such repo set, add it
	if !hasRepoSet {
		c.RepoSets = append(c.RepoSets, rs)
	}

	return nil
}

// Save writes the config back to the local filesystem
func (c *Config) Save() error {
	path, err := configPath()
	if err != nil {
		return err
	}

	out, err := json.Marshal(c)
	if err != nil {
		return err
	}

	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return err
	}

	err = os.WriteFile(path, out, 0600)
	if err != nil {
		return err
	}

	return nil
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
