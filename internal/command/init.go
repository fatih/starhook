package command

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"path/filepath"

	"github.com/fatih/starhook/internal/config"
	"github.com/peterbourgon/ff/v3/ffcli"
)

// Init is the config for the list subcommand, including a reference to the
// global config, for access to global flags.
type Init struct {
	rootConfig *Config
	out        io.Writer

	token string
	dir   string
	query string

	force bool
}

func initCmd(rootConfig *Config, out io.Writer) *ffcli.Command {
	cfg := Init{
		rootConfig: rootConfig,
		out:        out,
	}

	fs := flag.NewFlagSet("starhook init", flag.ExitOnError)
	fs.StringVar(&cfg.token, "token", "", "github token, i.e: GITHUB_TOKEN")
	fs.StringVar(&cfg.dir, "", "", "absolute path to download the repositories")
	fs.StringVar(&cfg.query, "query", "", "query to fetch the repositories")
	fs.BoolVar(&cfg.force, "force", false, "force initiliaze starhook")

	rootConfig.RegisterFlags(fs)

	return &ffcli.Command{
		Name:       "init",
		ShortUsage: "starhook init [flags] [<prefix>]",
		ShortHelp:  "Initialize starhook",
		FlagSet:    fs,
		Exec:       cfg.Exec,
	}
}

// Exec function for this command.
func (i *Init) Exec(ctx context.Context, _ []string) error {
	if i.token == "" {
		return errors.New("--token should be set")
	}
	if i.query == "" {
		return errors.New("--query should be set")
	}
	if i.dir == "" {
		return errors.New("--dir should be set")
	}

	if !filepath.IsAbs(i.dir) {
		return fmt.Errorf("--dir %q should be an absolute path", i.dir)
	}

	newCfg := &config.Config{
		Token:    i.token,
		Query:    i.query,
		ReposDir: i.dir,
	}

	err := config.Write(newCfg, i.force)
	if err != nil {
		return err
	}

	fmt.Fprintln(i.out, "starhook is initialized. Please run 'starhook sync' to download and sync your repositories.")
	return nil
}
