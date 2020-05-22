package gh

import (
	"context"

	"github.com/google/go-github/v28/github"
	"golang.org/x/oauth2"
)

type searchService interface {
	// Repositories searches repositories via various criteria.
	Repositories(ctx context.Context, query string, opt *github.SearchOptions) (*github.RepositoriesSearchResult, *github.Response, error)
}

// Client is responsible of searching and cloning the repositories
type Client struct {
	Search searchService
}

func NewClient(ctx context.Context, token string) *Client {
	ts := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: token},
	)

	tc := oauth2.NewClient(ctx, ts)
	ghClient := github.NewClient(tc)

	return &Client{
		Search: ghClient.Search,
	}
}

// FetchRepos fetches the repositories for the given query
func (c *Client) FetchRepos(ctx context.Context, query string) ([]github.Repository, error) {

	opts := &github.SearchOptions{
		ListOptions: github.ListOptions{PerPage: 50},
		Sort:        "updated",
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
