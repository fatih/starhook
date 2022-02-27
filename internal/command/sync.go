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

	log.Printf("[DEBUG] selected reposet: %s query: %s filters: %s\n", rs.Name, rs.Query, rs.Filter)
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

	log.Printf("[DEBUG] before filtering %d repos from GitHub\n", len(ghRepos))
	fetchedRepos := filterRepos(ghRepos, rs.Filter)
	log.Printf("[DEBUG] after filtering %d repos from GitHub\n", len(fetchedRepos))

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
	syncRepos, err := svc.SyncRepos(ctx, currentRepos, fetchedRepos)
	if err != nil {
		return err
	}

	total := len(syncRepos.Clone) + len(syncRepos.Update) + len(syncRepos.Delete)
	if total == 0 {
		log.Printf("everything is up-to-date")
		return nil
	}

	for _, r := range syncRepos.Clone {
		log.Printf("[DEBUG]  cloning: %q", r.Nwo)
	}
	for _, r := range syncRepos.Update {
		log.Printf("[DEBUG] updating: %q", r.Nwo)
	}
	for _, r := range syncRepos.Delete {
		log.Printf("[DEBUG] Deleting: %q", r.Nwo)
	}

	log.Printf("updates found:  \n")
	log.Printf("  clone  : %3d\n", len(syncRepos.Clone))
	log.Printf("  update : %3d\n", len(syncRepos.Update))
	log.Printf("  delete : %3d\n", len(syncRepos.Delete))

	if c.dryRun {
		log.Println("\nremove the '--dry-run' flag to sync the repositories")
		return nil
	}

	start := time.Now()
	if err := svc.CloneRepos(ctx, syncRepos.Clone); err != nil {
		return err
	}
	log.Printf("cloned: %d repositories (elapsed time: %s)\n",
		len(syncRepos.Clone), time.Since(start).String())

	start = time.Now()
	if err := svc.UpdateRepos(ctx, syncRepos.Update); err != nil {
		return err
	}
	log.Printf("updated: %d repositories (elapsed time: %s)\n",
		len(syncRepos.Update), time.Since(start).String())

	start = time.Now()
	if err := svc.DeleteRepos(ctx, syncRepos.Delete); err != nil {
		return err
	}
	log.Printf("deleted: %d repositories (elapsed time: %s)\n",
		len(syncRepos.Delete), time.Since(start).String())

	for _, repo := range syncRepos.Update {
		log.Printf("  %q is updated (last updated: %s)\n",
			repo.Name, humanize.Time(repo.SyncedAt))
	}

	return nil
}

func filterRepos(rps []*github.Repository, rules *config.FilterRules) []*internal.Repository {
	repos := make([]*internal.Repository, 0, len(rps))

	includedRepos := make(map[string]bool)
	excludedRepos := make(map[string]bool)
	if rules != nil {
		for _, nwo := range rules.Include {
			includedRepos[nwo] = true
		}

		for _, nwo := range rules.Exclude {
			excludedRepos[nwo] = true
		}
	}

	log.Printf("[DEBUG] rules: %+v included repos %+v, excluded repos %+v\n", rules, includedRepos, excludedRepos)

	var include func(string) bool
	if rules == nil {
		// if there are no rules, it means that we include all repositories
		include = func(nwo string) bool { return true }
	}

	if len(excludedRepos) != 0 {
		// we include all repos, expect the ones in the excluded list
		include = func(nwo string) bool { return !excludedRepos[nwo] }
	}

	if len(includedRepos) != 0 {
		// if included repos is set, it takes precedent over exclusion. In this
		// case, we only include repos that are in the list and exclude
		// everything else.
		include = func(nwo string) bool { return includedRepos[nwo] }
	}

	for _, repo := range rps {
		owner := repo.GetOwner().GetLogin()
		name := repo.GetName()
		nwo := fmt.Sprintf("%s/%s", owner, name)

		if include(nwo) {
			repos = append(repos, &internal.Repository{
				Nwo:    nwo,
				Owner:  owner,
				Name:   name,
				Branch: repo.GetDefaultBranch(),
			})
		}
	}

	return repos
}
