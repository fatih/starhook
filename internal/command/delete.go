package command

import (
	"context"
	"flag"
	"io"

	"github.com/peterbourgon/ff/v3/ffcli"
)

// Delete is the config for the list subcommand, including a reference to the
// global config, for access to global flags.
type Delete struct {
	rootConfig *Config
	out        io.Writer

	id int64
}

// New creates a new ffcli.Command for the list subcommand.
func deleteCmd(rootConfig *Config, out io.Writer) *ffcli.Command {
	cfg := Delete{
		rootConfig: rootConfig,
		out:        out,
	}

	fs := flag.NewFlagSet("starhook delete", flag.ExitOnError)
	fs.Int64Var(&cfg.id, "id", 0, "repository id to delete")

	rootConfig.RegisterFlags(fs)

	return &ffcli.Command{
		Name:       "delete",
		ShortUsage: "starhook delete [flags] [<prefix>]",
		ShortHelp:  "Delete available repositories",
		FlagSet:    fs,
		Exec:       cfg.Exec,
	}
}

// Exec function for this command.
func (c *Delete) Exec(ctx context.Context, _ []string) error {
	svc, err := newStarHookService()
	if err != nil {
		return err
	}

	return svc.DeleteRepo(ctx, c.id)
}
