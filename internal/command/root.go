package command

import (
	"context"
	"flag"
	"os"

	"github.com/fatih/starhook/internal/config"
	"github.com/fatih/starhook/internal/gh"
	"github.com/fatih/starhook/internal/jsonstore"
	"github.com/fatih/starhook/internal/starhook"

	"github.com/peterbourgon/ff/v3"
	"github.com/peterbourgon/ff/v3/ffcli"
)

// Config for the root command, including flags and types that should be
// available to each subcommand.
type Config struct {
	Verbose bool

	Service *starhook.Service
}

func Run() error {
	ctx := context.Background()

	var (
		out                     = os.Stdout
		rootCommand, rootConfig = newRootCommand()
	)

	rootCommand.Subcommands = []*ffcli.Command{
		listCmd(rootConfig, out),
		deleteCmd(rootConfig, out),
		syncCmd(rootConfig, out),
		initCmd(rootConfig, out),
	}

	return rootCommand.ParseAndRun(ctx, os.Args[1:])
}

// newRootCommand constructs a usable ffcli.Command and an empty Config. The config's token
// and verbose fields will be set after a successful parse. The caller must
// initialize the config's object API client field.
func newRootCommand() (*ffcli.Command, *Config) {
	var cfg Config

	fs := flag.NewFlagSet("starhook", flag.ExitOnError)
	cfg.RegisterFlags(fs)

	return &ffcli.Command{
		Name:       "starhook",
		ShortUsage: "starhook [flags] <subcommand> [flags] [<arg>...]",
		FlagSet:    fs,
		Exec:       cfg.Exec,
		Options: []ff.Option{
			ff.WithEnvVarPrefix("STARHOOK"),
			ff.WithConfigFileFlag("config"),
			ff.WithConfigFileParser(ff.JSONParser),
		},
	}, &cfg
}

// RegisterFlags registers the flag fields into the provided flag.FlagSet. This
// helper function allows subcommands to register the root flags into their
// flagsets, creating "global" flags that can be passed after any subcommand at
// the commandline.
func (c *Config) RegisterFlags(fs *flag.FlagSet) {
	fs.BoolVar(&c.Verbose, "v", false, "log verbose output")
	_ = fs.String("config", "", "config file (optional)")
}

// Exec function for this command.
func (c *Config) Exec(context.Context, []string) error {
	// The root command has no meaning, so if it gets executed,
	// display the usage text to the user instead.
	return flag.ErrHelp
}

func newStarHookService() (*starhook.Service, error) {
	cfg, err := config.Load()
	if err != nil {
		return nil, err
	}

	ctx := context.Background()
	ghClient := gh.NewClient(ctx, cfg.Token)

	store, err := jsonstore.NewRepositoryStore(cfg.ReposDir, cfg.Query)
	if err != nil {
		return nil, err
	}

	return starhook.NewService(ghClient, store, cfg.ReposDir), nil
}
