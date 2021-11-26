package command

import (
	"context"
	"flag"
	"fmt"
	"io"

	"github.com/fatih/starhook/internal"
	"github.com/fatih/starhook/internal/config"
	"github.com/fatih/starhook/internal/gh"
	"github.com/fatih/starhook/internal/jsonstore"
	"github.com/fatih/starhook/internal/starhook"
	"github.com/google/go-github/v39/github"
	"github.com/peterbourgon/ff/v3/ffcli"
)

// Sync is the config for the list subcommand, including a reference to the
// global config, for access to global flags.
type Sync struct {
	rootConfig *Config
	out        io.Writer

	dryRun bool
}

func syncCmd(rootConfig *Config, out io.Writer) *ffcli.Command {
	cfg := Sync{
		rootConfig: rootConfig,
		out:        out,
	}

	fs := flag.NewFlagSet("starhook sync", flag.ExitOnError)
	fs.BoolVar(&cfg.dryRun, "dry-run", false, "dry-run the given action")

	rootConfig.RegisterFlags(fs)

	return &ffcli.Command{
		Name:       "sync",
		ShortUsage: "starhook sync [flags] [<prefix>]",
		ShortHelp:  "Sync available repositories",
		FlagSet:    fs,
		Exec:       cfg.Exec,
	}
}

// Exec function for this command.
func (c *Sync) Exec(ctx context.Context, _ []string) error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}

	ghClient := gh.NewClient(ctx, cfg.Token)

	store, err := jsonstore.NewRepositoryStore(cfg.ReposDir, cfg.Query)
	if err != nil {
		return err
	}

	svc := starhook.NewService(ghClient, store, cfg.ReposDir)

	fmt.Fprintln(c.out, "==> querying for latest repositories ...")
	ghRepos, err := ghClient.FetchRepos(ctx, cfg.Query)
	if err != nil {
		return err
	}
	fetchedRepos := toRepos(ghRepos)

	if err := svc.SyncRepos(ctx, fetchedRepos); err != nil {
		return err
	}

	clone, update, err := svc.ReposToUpdate(ctx)
	if err != nil {
		return err
	}
	total := len(clone) + len(update)
	if total == 0 {
		fmt.Fprintf(c.out, "==> everything is up-to-date")
		return nil
	}

	fmt.Fprintf(c.out, "==> updates found:  \n")
	fmt.Fprintf(c.out, "  clone  : %3d\n", len(clone))
	fmt.Fprintf(c.out, "  update : %3d\n", len(update))

	if c.dryRun {
		fmt.Fprintln(c.out, "\nremove the '--dry-run' flag to update & clone the repositories")
		return nil
	}

	if err := svc.CloneRepos(ctx, clone); err != nil {
		return err
	}
	if err := svc.UpdateRepos(ctx, update); err != nil {
		return err
	}

	return nil
}

func toRepos(rps []*github.Repository) []*internal.Repository {
	repos := make([]*internal.Repository, 0, len(rps))
	for _, repo := range rps {
		owner := repo.GetOwner().GetLogin()
		name := repo.GetName()

		repos = append(repos, &internal.Repository{
			Nwo:    fmt.Sprintf("%s/%s", owner, name),
			Owner:  owner,
			Name:   name,
			Branch: repo.GetDefaultBranch(),
		})
	}

	return repos
}
