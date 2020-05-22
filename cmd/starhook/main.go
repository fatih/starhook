package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"os"

	"github.com/fatih/starhook/internal/gh"
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
		update = flag.Bool("update", false, "update the repositores to latest HEAD")
	)
	flag.Parse()

	if *token == "" {
		return errors.New("GitHub API token is not set via --token")
	}

	ctx := context.Background()
	ghClient := gh.NewClient(ctx, *token)

	sh := starhook.NewClient(ghClient, *dir, *query, *update)

	return sh.Run(ctx)
}
