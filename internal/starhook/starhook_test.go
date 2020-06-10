package starhook

import (
	"context"
	"testing"

	"github.com/fatih/starhook/internal"
	"github.com/fatih/starhook/internal/mock"
	"github.com/fatih/starhook/internal/testutil"
)

func TestNewService(t *testing.T) {
	svc := NewService(nil, nil, "")
	testutil.Assert(t, svc != nil, "service should be not nil")
}

func TestService_ListRepos(t *testing.T) {
	ctx := context.Background()

	store := &mock.RepositoryStore{
		FindReposFn: func(ctx context.Context, filter internal.RepositoryFilter, opt internal.FindOptions) ([]*internal.Repository, error) {
			return []*internal.Repository{
				{ID: 1},
				{ID: 2},
			}, nil
		},
	}

	svc := NewService(nil, store, "")

	query := "org:github language:go"

	err := svc.ListRepos(ctx, query)
	testutil.Ok(t, err)
	testutil.Assert(t, store.FindReposInvoked, "FindRepos() should be called")
}
