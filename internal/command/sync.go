package command

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/fatih/starhook/internal"
	"github.com/fatih/starhook/internal/config"
	"github.com/fatih/starhook/internal/fsstore"
	"github.com/fatih/starhook/internal/gh"
	"github.com/fatih/starhook/internal/jsonstore"
	"github.com/fatih/starhook/internal/starhook"

	"github.com/dustin/go-humanize"
	"github.com/google/go-github/v39/github"
	"github.com/peterbourgon/ff/v3/ffcli"
)

// Sync is the config for the sync subcommand, including a reference to the
// global config, for access to global flags.
type Sync struct {
	rootConfig *RootConfig

	dryRun bool
	force  bool
}

func syncCmd(rootConfig *RootConfig) *ffcli.Command {
	cfg := Sync{
		rootConfig: rootConfig,
	}

	fs := flag.NewFlagSet("starhook sync", flag.ExitOnError)
	fs.BoolVar(&cfg.dryRun, "dry-run", false, "dry-run the given action")
	fs.BoolVar(&cfg.force, "force", false, "override existing repository directory")

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
	log.Println("[DEBUG] loading the configuration")

	cfg, err := config.Load()
	if err != nil {
		return err
	}

	rs, err := cfg.SelectedRepoSet()
	if err != nil {
		return err
	}

	log.Printf("[DEBUG] selected reposet: %s query: %s\n", rs.Name, rs.Query)
	ghClient := gh.NewClient(ctx, rs.Token)

	if err := os.MkdirAll(filepath.Dir(rs.ReposDir), 0700); err != nil {
		return err
	}

	log.Printf("[DEBUG] using repo dir: %s\n", rs.ReposDir)
	store, err := jsonstore.NewMetadataStore(rs.ReposDir, rs.Query)
	if err != nil {
		return err
	}

	fsStore, err := fsstore.NewRepositoryStore(rs.ReposDir)
	if err != nil {
		return err
	}

	svc := starhook.NewService(ghClient, store, fsStore)

	log.Println("querying for latest repositories ...")
	ghRepos, err := ghClient.FetchRepos(ctx, rs.Query)
	if err != nil {
		return err
	}
	fetchedRepos := toRepos(ghRepos)

	log.Println("[DEBUG] fetching repos from GitHub")
	currentRepos, err := svc.ListRepos(ctx)
	if err != nil {
		return err
	}

	lastSynced := time.Time{}
	for _, repo := range currentRepos {
		if repo.SyncedAt.After(lastSynced) {
			lastSynced = repo.SyncedAt
		}
	}

	log.Printf("last synced: %s\n", humanize.Time(lastSynced))

	log.Println("[DEBUG] syncing remote repos to local directory")
	if err := svc.SyncRepos(ctx, currentRepos, fetchedRepos); err != nil {
		return err
	}

	clone, update, err := svc.ReposToUpdate(ctx)
	if err != nil {
		return err
	}

	total := len(clone) + len(update)
	if total == 0 {
		log.Printf("everything is up-to-date")
		return nil
	}

	log.Printf("updates found:  \n")
	log.Printf("  clone  : %3d\n", len(clone))
	for _, r := range clone {
		log.Printf("[DEBUG]  cloning: %q", r.Nwo)
	}
	log.Printf("  update : %3d\n", len(update))
	for _, r := range update {
		log.Printf("[DEBUG] updating: %q", r.Nwo)
	}

	if c.dryRun {
		log.Println("\nremove the '--dry-run' flag to update & clone the repositories")
		return nil
	}

	start := time.Now()
	if err := svc.CloneRepos(ctx, clone); err != nil {
		return err
	}
	log.Printf("cloned: %d repositories (elapsed time: %s)\n",
		len(clone), time.Since(start).String())

	start = time.Now()
	if err := svc.UpdateRepos(ctx, update); err != nil {
		return err
	}

	for _, repo := range update {
		log.Printf("  %q is updated (last updated: %s)\n",
			repo.Name, humanize.Time(repo.SyncedAt))
	}
	log.Printf("updated: %d repositories (elapsed time: %s)\n",
		len(update), time.Since(start).String())

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
