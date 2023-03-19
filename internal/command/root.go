package command

import (
	"context"
	"flag"
	"io"
	"log"
	"os"

	"github.com/fatih/starhook/internal/config"
	"github.com/fatih/starhook/internal/fsstore"
	"github.com/fatih/starhook/internal/gh"
	"github.com/fatih/starhook/internal/jsonstore"
	"github.com/fatih/starhook/internal/starhook"

	"github.com/hashicorp/logutils"
	"github.com/peterbourgon/ff/v3"
	"github.com/peterbourgon/ff/v3/ffcli"
)

// RootConfig for the root command, including flags and types that should be
// available to each subcommand.
type RootConfig struct {
	Verbose bool

	Service *starhook.Service

	out io.Writer
}

func Run() error {
	ctx := context.Background()

	rootCommand, rootConfig := newRootCommand()

	rootCommand.Subcommands = []*ffcli.Command{
		configCmd(rootConfig),
		listCmd(rootConfig),
		syncCmd(rootConfig),
	}

	if err := rootCommand.Parse(os.Args[1:]); err != nil {
		return err
	}

	filter := &logutils.LevelFilter{
		Levels:   []logutils.LogLevel{"DEBUG", "INFO", "WARN", "ERROR"},
		MinLevel: logutils.LogLevel("INFO"),
		Writer:   rootConfig.out,
	}

	if rootConfig.Verbose {
		filter.MinLevel = logutils.LogLevel("DEBUG")
	} else {
		// don't show time in non verbose mode
		log.SetFlags(0)
	}
	log.SetOutput(filter)

	return rootCommand.Run(ctx)
}

// newRootCommand constructs a usable ffcli.Command and an empty Config. The config's token
// and verbose fields will be set after a successful parse. The caller must
// initialize the config's object API client field.
func newRootCommand() (*ffcli.Command, *RootConfig) {
	cfg := &RootConfig{
		// NOTE(fatih) should we make this configurable?
		out: os.Stdout,
	}

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
	}, cfg
}

// RegisterFlags registers the flag fields into the provided flag.FlagSet. This
// helper function allows subcommands to register the root flags into their
// flagsets, creating "global" flags that can be passed after any subcommand at
// the commandline.
func (c *RootConfig) RegisterFlags(fs *flag.FlagSet) {
	fs.BoolVar(&c.Verbose, "v", false, "log verbose output")
	_ = fs.String("config", "", "config file (optional)")
}

// Exec function for this command.
func (c *RootConfig) Exec(context.Context, []string) error {
	// The root command has no meaning, so if it gets executed,
	// display the usage text to the user instead.
	return flag.ErrHelp
}

func newStarHookService() (*starhook.Service, error) {
	cfg, err := config.Load()
	if err != nil {
		return nil, err
	}

	rs, err := cfg.SelectedRepoSet()
	if err != nil {
		return nil, err
	}

	ctx := context.Background()
	ghClient := gh.NewClient(ctx, cfg.GitHubToken())

	store, err := jsonstore.NewMetadataStore(rs.ReposDir, rs.Query)
	if err != nil {
		return nil, err
	}

	fsStore, err := fsstore.NewRepositoryStore(rs.ReposDir)
	if err != nil {
		return nil, err
	}

	return starhook.NewService(ghClient, store, fsStore), nil
}
