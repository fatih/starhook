package gh

import (
	"context"
	"testing"
	"time"

	"github.com/fatih/starhook/internal/testutil"
	"github.com/google/go-github/v28/github"
)

func TestNewClient(t *testing.T) {
	ctx := context.Background()
	client := NewClient(ctx, "")
	testutil.Assert(t, client != nil, "client should be not nil")
}

func TestClient_FetchRepos(t *testing.T) {
	ctx := context.Background()

	repoName1 := "foo"
	repoName2 := "bar"

	searchService := &mockSearchService{
		RepositoriesFunc: func(ctx context.Context, query string, opt *github.SearchOptions) (*github.RepositoriesSearchResult, *github.Response, error) {
			return &github.RepositoriesSearchResult{
				Repositories: []github.Repository{
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
	testutil.Ok(t, err)
	testutil.Assert(t, searchService.RepositoriesInvoked, "Repositories() should be called")
	testutil.Equals(t, len(repos), 2)
}

func TestClient_Branch(t *testing.T) {
	ctx := context.Background()

	wantSHA := "123"
	updatedAt := time.Now()

	repoService := &mockRepositoriesService{
		GetBranchFunc: func(ctx context.Context, owner, repo, branch string) (*github.Branch, *github.Response, error) {
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
	testutil.Ok(t, err)
	testutil.Equals(t, resp.SHA, wantSHA)
	testutil.Equals(t, resp.UpdatedAt, updatedAt)
	testutil.Assert(t, repoService.GetBranchInvoked, "GetBranch() should be called")
}

type mockRepositoriesService struct {
	GetBranchFunc    func(ctx context.Context, owner, repo, branch string) (*github.Branch, *github.Response, error)
	GetBranchInvoked bool
}

func (m *mockRepositoriesService) GetBranch(ctx context.Context, owner, repo, branch string) (*github.Branch, *github.Response, error) {
	m.GetBranchInvoked = true
	if m.GetBranchFunc != nil {
		return m.GetBranchFunc(ctx, owner, repo, branch)
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
