package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"os"

	"github.com/fatih/starhook/internal/gh"
	"github.com/fatih/starhook/internal/jsonstore"
	"github.com/fatih/starhook/internal/starhook"
)

func main() {
	if err := realMain(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func realMain() error {
	var (
		token  = flag.String("token", "", "github token, i.e: GITHUB_TOKEN")
		dir    = flag.String("dir", "repos", "path to download the repositories")
		query  = flag.String("query", "org:github language:go", "query to fetch")
		sync   = flag.Bool("sync", false, "sync the repositores to the given query")
		fetch  = flag.Bool("fetch", false, "fetch the repositores for the given query")
		list   = flag.Bool("list", false, "list the repositores for the given query")
		dryRun = flag.Bool("dry-run", false, "dry-run the given action")
	)
	flag.Parse()

	if *token == "" {
		return errors.New("GitHub API token is not set via --token")
	}

	ctx := context.Background()
	ghClient := gh.NewClient(ctx, *token)
	store, err := jsonstore.NewRepositoryStore(*dir)
	if err != nil {
		return err
	}

	svc := starhook.NewService(ghClient, store, *dir)

	if *fetch {
		clone, update, err := svc.ReposToUpdate(ctx, *query)
		if err != nil {
			return err
		}

		fmt.Printf("==> found %d repositories:\n", len(clone)+len(update))
		fmt.Printf("  new   : %3d\n", len(clone))
		fmt.Printf("  update: %3d\n", len(update))
		if *dryRun {
			fmt.Println("\nremove -dry-run to fetch & clone the repositories")
			return nil
		}

		if err := svc.CloneRepos(ctx, clone); err != nil {
			return err
		}
		if err := svc.UpdateRepos(ctx, update); err != nil {
			return err
		}
		return nil
	} else if *sync {
		return svc.SyncRepos(ctx, *query)
	} else if *list {
		return svc.ListRepos(ctx, *query)
	} else {
		return errors.New("please provide an option: -fetch, -sync or -list")
	}
}
