package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/google/go-github/v28/github"
	"golang.org/x/oauth2"
	"golang.org/x/sync/errgroup"
	"golang.org/x/sync/semaphore"
)

func main() {
	if err := realMain(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

type searchService interface {
	// Repositories searches repositories via various criteria.
	Repositories(ctx context.Context, query string, opt *github.SearchOptions) (*github.RepositoriesSearchResult, *github.Response, error)
}

// Client is responsible of searching and cloning the repositories
type Client struct {
	Search   searchService
	CloneDir string
}

func realMain() error {
	var (
		token = flag.String("token", "", "github token, i.e: GITHUB_TOKEN")
		dir   = flag.String("dir", "repos", "path to download the repositories")
		query = flag.String("query", "org:github language:go", "query to fetch")
	)
	flag.Parse()

	if *token == "" {
		return errors.New("GitHub API token is not set via --token")
	}

	fmt.Println("==> searching and fetching repositories")
	start := time.Now()
	ctx := context.Background()
	ts := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: *token},
	)
	tc := oauth2.NewClient(ctx, ts)
	ghClient := github.NewClient(tc)

	client := Client{
		Search:   ghClient.Search,
		CloneDir: *dir,
	}

	var repos []github.Repository
	repodb := filepath.Join(*dir, "repos.json")
	if out, err := ioutil.ReadFile(repodb); err != nil {
		repos, err = client.fetchRepos(ctx, *query)
		if err != nil {
			return err
		}

		// dump data so we don't fetch it again
		out, err := json.MarshalIndent(repos, " ", " ")
		if err != nil {
			return err
		}

		if err := ioutil.WriteFile(repodb, out, 0644); err != nil {
			return err
		}

		fmt.Printf("==> fetched: %d repositores (elapsed time: %s)\n",
			len(repos), time.Since(start).String())
	} else {
		// load from cached file
		if err := json.Unmarshal(out, &repos); err != nil {
			return err
		}

		fmt.Printf("==> using existing data: %d repositores (elapsed time: %s)\n",
			len(repos), time.Since(start).String())
	}

	if err := client.cloneRepos(ctx, repos); err != nil {
		return err
	}

	return nil
}

func (c *Client) fetchRepos(ctx context.Context, query string) ([]github.Repository, error) {
	opts := &github.SearchOptions{
		ListOptions: github.ListOptions{PerPage: 50},
	}

	var repos []github.Repository
	for {
		res, resp, err := c.Search.Repositories(ctx, query, opts)
		if err != nil {
			return nil, err
		}

		repos = append(repos, res.Repositories...)

		if resp.NextPage == 0 {
			break
		}

		opts.Page = resp.NextPage
	}

	return repos, nil
}

func (c *Client) cloneRepos(ctx context.Context, repos []github.Repository) error {
	if _, err := exec.LookPath("git"); err != nil {
		// make sure that `git` exists before we continue
		return errors.New("couldn't find 'git' in PATH")
	}

	fmt.Println("==> cloning repositories")
	start := time.Now()

	// download at max 10 repos at the same time to not overload and burst the
	// server. Also makes it easier
	const maxWorkers = 10
	sem := semaphore.NewWeighted(maxWorkers)

	g, ctx := errgroup.WithContext(ctx)
	for _, repo := range repos {
		repo := repo

		err := sem.Acquire(ctx, 1)
		if err != nil {
			fmt.Printf("acquire err = %+v\n", err)
			break
		}

		g.Go(func() error {
			defer sem.Release(1)
			return c.cloneRepo(ctx, repo)
		})
	}

	if err := g.Wait(); err != nil {
		fmt.Printf("g.Wait() err = %+v\n", err)
	}

	fmt.Printf("==> cloned: %d repositores (elapsed time: %s)\n",
		len(repos), time.Since(start).String())
	return nil
}

func (c *Client) cloneRepo(ctx context.Context, repo github.Repository) error {
	repoDir := filepath.Join(c.CloneDir, repo.GetName())

	// do not clone if it exists
	if _, err := os.Stat(repoDir); err == nil {
		return nil
	}

	fmt.Printf("  cloning %s\n", repo.GetName())
	args := []string{"clone", repo.GetCloneURL(), "--depth=1", repoDir}
	cmd := exec.Command("git", args...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("git clone failed for url: %q err: %s out: %s",
			repo.GetCloneURL(), err, string(out))
	}

	return nil
}
