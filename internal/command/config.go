package command

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"text/tabwriter"

	"github.com/fatih/starhook/internal/config"
	"github.com/lucasepe/codename"
	"github.com/peterbourgon/ff/v3/ffcli"
)

// Config is the config for the list subcommand, including a reference to the
// global config, for access to global flags.
type Config struct {
	rootConfig *RootConfig
}

// New creates a new ffcli.Command for the list subcommand.
func configCmd(rootConfig *RootConfig) *ffcli.Command {
	cfg := Config{
		rootConfig: rootConfig,
	}

	fs := flag.NewFlagSet("starhook config", flag.ExitOnError)
	rootConfig.RegisterFlags(fs)

	// TODO
	// following subcommands need to be added
	// config delete // delete a configuration and all repositories (ask for confirmation, it's destructable)
	// config set key value   // update existing value, only for token and query

	return &ffcli.Command{
		Name:       "config",
		ShortUsage: "starhook config [flags] [<prefix>]",
		ShortHelp:  "Manage existing configurations",
		FlagSet:    fs,
		Exec:       cfg.Exec,
		Subcommands: []*ffcli.Command{
			configAddCmd(rootConfig),
			configShowCmd(rootConfig),
			configListCmd(rootConfig),
			configSwitchCmd(rootConfig),
		},
	}
}

// Exec function for this command.
func (c *Config) Exec(ctx context.Context, _ []string) error {
	return flag.ErrHelp
}

func configAddCmd(rootConfig *RootConfig) *ffcli.Command {
	var (
		name  string // optional
		token string
		dir   string
		query string

		force bool
	)

	fs := flag.NewFlagSet("starhook config add", flag.ExitOnError)
	rootConfig.RegisterFlags(fs)

	fs.StringVar(&token, "token", "", "github token, i.e: GITHUB_TOKEN")
	fs.StringVar(&dir, "dir", "", "absolute path to download the repositories")
	fs.StringVar(&query, "query", "", "query to fetch the repositories")
	fs.StringVar(&name, "name", "", "name of the configuration (optional)")
	fs.BoolVar(&force, "force", false, "override existing configuration for a given --name ")

	return &ffcli.Command{
		Name:       "add",
		ShortUsage: "starhook config add [flags] [<prefix>]",
		ShortHelp:  "Add a new configuration",
		FlagSet:    fs,
		Exec: func(ctx context.Context, _ []string) error {
			if token == "" {
				return errors.New("--token should be set")
			}
			if query == "" {
				return errors.New("--query should be set")
			}
			if dir == "" {
				return errors.New("--dir should be set")
			}

			if !filepath.IsAbs(dir) {
				return fmt.Errorf("--dir %q should be an absolute path", dir)
			}

			name := name
			if name == "" {
				rng, err := codename.DefaultRNG()
				if err != nil {
					return err
				}
				name = codename.Generate(rng, 0)
			}

			rs := &config.RepoSet{
				Name:     name,
				Token:    token,
				Query:    query,
				ReposDir: dir,
			}

			cfg, err := config.Load()
			if err != nil {
				if errors.Is(err, os.ErrNotExist) {
					cfg, err = config.Create()
					if err != nil {
						return err
					}
				} else {
					return err
				}
			}

			err = cfg.AddRepoSet(rs, force)
			if err != nil {
				return err
			}

			err = cfg.Save()
			if err != nil {
				return err
			}

			log.Printf("starhook is initialized (config name: %q)\n\nPlease run 'starhook config switch %s && starhook sync' to download and sync your repositories.\n", name, name)
			return nil
		},
	}
}

func configShowCmd(rootConfig *RootConfig) *ffcli.Command {
	fs := flag.NewFlagSet("starhook config show", flag.ExitOnError)
	rootConfig.RegisterFlags(fs)

	return &ffcli.Command{
		Name:       "show",
		ShortUsage: "starhook config show [flags] [<prefix>]",
		ShortHelp:  "Show selected configuration",
		FlagSet:    fs,
		Exec: func(ctx context.Context, _ []string) error {
			cfg, err := config.Load()
			if err != nil {
				return err
			}

			rs, err := cfg.SelectedRepoSet()
			if err != nil {
				return err
			}

			const padding = 3
			w := tabwriter.NewWriter(rootConfig.out, 0, 0, padding, ' ', 0)

			fmt.Fprintf(w, "Name\t%+v\n", rs.Name)
			fmt.Fprintf(w, "Query\t%+v\n", rs.Query)
			fmt.Fprintf(w, "Repositories Directory\t%+v\n", rs.ReposDir)
			w.Flush()

			return nil
		},
	}
}

func configListCmd(rootConfig *RootConfig) *ffcli.Command {
	fs := flag.NewFlagSet("starhook config list", flag.ExitOnError)
	rootConfig.RegisterFlags(fs)

	return &ffcli.Command{
		Name:       "list",
		ShortUsage: "starhook config list [flags] [<prefix>]",
		ShortHelp:  "List existings configurations",
		FlagSet:    fs,
		Exec: func(ctx context.Context, _ []string) error {
			cfg, err := config.Load()
			if err != nil {
				return err
			}

			const padding = 3
			w := tabwriter.NewWriter(rootConfig.out, 0, 0, padding, ' ', 0)

			for _, rs := range cfg.RepoSets {
				fmt.Fprintf(w, "Name\t%+v\n", rs.Name)
				fmt.Fprintf(w, "Query\t%+v\n", rs.Query)
				fmt.Fprintf(w, "Repositories Directory\t%+v\n\n", rs.ReposDir)
			}
			w.Flush()

			return nil
		},
	}
}

func configSwitchCmd(rootConfig *RootConfig) *ffcli.Command {
	fs := flag.NewFlagSet("starhook config switch", flag.ExitOnError)
	rootConfig.RegisterFlags(fs)

	return &ffcli.Command{
		Name:       "switch",
		ShortUsage: "starhook config switch [<config_name>]",
		ShortHelp:  "Select a different configuration",
		FlagSet:    fs,
		Exec: func(ctx context.Context, args []string) error {
			if len(args) == 0 {
				return flag.ErrHelp
			}

			configName := args[0]

			cfg, err := config.Load()
			if err != nil {
				return err
			}

			hasConfig := false
			for _, rs := range cfg.RepoSets {
				if rs.Name == configName {
					hasConfig = true
				}
			}

			if !hasConfig {
				return fmt.Errorf("config name %q does not exist", configName)
			}

			cfg.Selected = configName

			err = cfg.Save()
			if err != nil {
				return err
			}

			log.Printf("Switched to %q\n", configName)
			return nil
		},
	}
}
