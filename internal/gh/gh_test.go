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

	wantSHA := "123"
	updatedAt := time.Now()

	client := &Client{
		Repositories: &mockRepositoriesService{
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
		},
	}

	owner := "fatih"
	repo := "vim-go"
	branch := "main"

	resp, err := client.Branch(ctx, owner, repo, branch)
	testutil.Ok(t, err)
	testutil.Equals(t, resp.SHA, wantSHA)
	testutil.Equals(t, resp.UpdatedAt, updatedAt)
}

type mockRepositoriesService struct {
	GetBranchFunc    func(ctx context.Context, owner, repo, branch string) (*github.Branch, *github.Response, error)
	GetBranchInvoked bool
}

func (m *mockRepositoriesService) GetBranch(ctx context.Context, owner, repo, branch string) (*github.Branch, *github.Response, error) {
	if m.GetBranchFunc != nil {
		return m.GetBranchFunc(ctx, owner, repo, branch)
	}
	return nil, &github.Response{}, nil
}
