package gh

import (
	"context"
	"testing"
	"time"

	"github.com/google/go-github/v39/github"

	qt "github.com/frankban/quicktest"
)

func TestNewClient(t *testing.T) {
	c := qt.New(t)
	ctx := context.Background()
	client := NewClient(ctx, "")
	c.Assert(client, qt.Not(qt.IsNil), qt.Commentf("client should be not nil"))
}

func TestClient_FetchRepos(t *testing.T) {
	c := qt.New(t)
	ctx := context.Background()

	repoName1 := "foo"
	repoName2 := "bar"

	searchService := &mockSearchService{
		RepositoriesFunc: func(ctx context.Context, query string, opt *github.SearchOptions) (*github.RepositoriesSearchResult, *github.Response, error) {
			return &github.RepositoriesSearchResult{
				Repositories: []*github.Repository{
					{
						Name: &repoName1,
					},
					{

						Name: &repoName2,
					},
				},
			}, &github.Response{}, nil
		},
	}

	client := &Client{
		Search: searchService,
	}

	query := "org:github language:go"

	repos, err := client.FetchRepos(ctx, query)
	c.Assert(err, qt.IsNil)
	c.Assert(searchService.RepositoriesInvoked, qt.IsTrue, qt.Commentf("Repositories() should be called"))
	c.Assert(repos, qt.HasLen, 2)
}

func TestClient_Branch(t *testing.T) {
	c := qt.New(t)
	ctx := context.Background()

	wantSHA := "123"
	updatedAt := time.Now()

	repoService := &mockRepositoriesService{
		GetBranchFunc: func(ctx context.Context, owner, repo, branch string, followRedirects bool) (*github.Branch, *github.Response, error) {

			c.Assert(followRedirects, qt.IsTrue)

			return &github.Branch{
				Commit: &github.RepositoryCommit{
					SHA: &wantSHA,
					Commit: &github.Commit{
						Committer: &github.CommitAuthor{
							Date: &updatedAt,
						},
					},
				},
			}, nil, nil
		},
	}

	client := &Client{
		Repositories: repoService,
	}

	owner := "fatih"
	repo := "vim-go"
	branch := "main"

	resp, err := client.Branch(ctx, owner, repo, branch)
	c.Assert(err, qt.IsNil)
	c.Assert(resp.SHA, qt.Equals, wantSHA)
	c.Assert(resp.UpdatedAt, qt.Equals, updatedAt)
	c.Assert(repoService.GetBranchInvoked, qt.IsTrue, qt.Commentf("GetBranch() should be called"))
}

type mockRepositoriesService struct {
	GetBranchFunc    func(ctx context.Context, owner, repo, branch string, followRedirects bool) (*github.Branch, *github.Response, error)
	GetBranchInvoked bool
}

func (m *mockRepositoriesService) GetBranch(ctx context.Context, owner, repo, branch string, followRedirects bool) (*github.Branch, *github.Response, error) {
	m.GetBranchInvoked = true
	if m.GetBranchFunc != nil {
		return m.GetBranchFunc(ctx, owner, repo, branch, followRedirects)
	}
	return nil, &github.Response{}, nil
}

type mockSearchService struct {
	RepositoriesFunc    func(ctx context.Context, query string, opt *github.SearchOptions) (*github.RepositoriesSearchResult, *github.Response, error)
	RepositoriesInvoked bool
}

func (m *mockSearchService) Repositories(ctx context.Context, query string, opt *github.SearchOptions) (*github.RepositoriesSearchResult, *github.Response, error) {
	m.RepositoriesInvoked = true
	if m.RepositoriesFunc != nil {
		return m.RepositoriesFunc(ctx, query, opt)
	}
	return nil, &github.Response{}, nil
}
