package gh

import (
	"context"
	"time"

	"github.com/google/go-github/v39/github"
	"golang.org/x/oauth2"
)

type Branch struct {
	SHA       string
	UpdatedAt time.Time
}

type searchService interface {
	// Repositories searches repositories via various criteria.
	Repositories(ctx context.Context, query string, opt *github.SearchOptions) (*github.RepositoriesSearchResult, *github.Response, error)
}

type repositoryService interface {
	// GetBranch gets the specified branch for a repository.
	GetBranch(ctx context.Context, owner, repo, branch string, followRedirects bool) (*github.Branch, *github.Response, error)
}

// Client is responsible of searching and cloning the repositories
type Client struct {
	Search       searchService
	Repositories repositoryService
}

func NewClient(ctx context.Context, token string) *Client {
	ts := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: token},
	)

	tc := oauth2.NewClient(ctx, ts)
	ghClient := github.NewClient(tc)

	return &Client{
		Search:       ghClient.Search,
		Repositories: ghClient.Repositories,
	}
}

// FetchRepos fetches the repositories for the given query
func (c *Client) FetchRepos(ctx context.Context, query string) ([]*github.Repository, error) {
	opts := &github.SearchOptions{
		ListOptions: github.ListOptions{PerPage: 50},
	}

	var repos []*github.Repository
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

func (c *Client) Branch(ctx context.Context, owner, name, branch string) (*Branch, error) {
	res, _, err := c.Repositories.GetBranch(ctx, owner, name, branch, true)
	if err != nil {
		return nil, err
	}

	updatedAt := res.GetCommit().GetCommit().GetCommitter().GetDate()
	sha := res.GetCommit().GetSHA()
	return &Branch{
		SHA:       sha,
		UpdatedAt: updatedAt,
	}, nil
}
