package command

import (
	"context"
	"flag"
	"fmt"
	"io"
	"time"

	"github.com/dustin/go-humanize"
	"github.com/peterbourgon/ff/v3/ffcli"
)

// List is the config for the list subcommand, including a reference to the
// global config, for access to global flags.
type List struct {
	rootConfig *RootConfig
	out        io.Writer

	withAccessTimes bool
}

// New creates a new ffcli.Command for the list subcommand.
func listCmd(rootConfig *RootConfig, out io.Writer) *ffcli.Command {
	cfg := List{
		rootConfig: rootConfig,
		out:        out,
	}

	fs := flag.NewFlagSet("starhook list", flag.ExitOnError)
	fs.BoolVar(&cfg.withAccessTimes, "a", false, "include last access time of each object")

	rootConfig.RegisterFlags(fs)

	return &ffcli.Command{
		Name:       "list",
		ShortUsage: "starhook list [flags] [<prefix>]",
		ShortHelp:  "List available repositories",
		FlagSet:    fs,
		Exec:       cfg.Exec,
	}
}

// Exec function for this command.
func (c *List) Exec(ctx context.Context, _ []string) error {
	svc, err := newStarHookService()
	if err != nil {
		return err
	}

	repos, err := svc.ListRepos(ctx)
	if err != nil {
		return err
	}

	lastUpdated := time.Time{}
	for _, repo := range repos {
		if repo.UpdatedAt.After(lastUpdated) {
			lastUpdated = repo.UpdatedAt
		}
		fmt.Printf("%3d %s\n", repo.ID, repo.Nwo)
	}

	fmt.Fprintf(c.out, "==> local %d repositories (last synced: %s)\n", len(repos), humanize.Time(lastUpdated))

	return nil

}
