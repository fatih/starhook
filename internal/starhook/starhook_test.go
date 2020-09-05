package starhook

import (
	"context"
	"testing"

	"github.com/fatih/starhook/internal"
	"github.com/fatih/starhook/internal/mock"

	qt "github.com/frankban/quicktest"
)

func TestNewService(t *testing.T) {
	c := qt.New(t)
	svc := NewService(nil, nil, "")
	c.Assert(svc, qt.Not(qt.IsNil), qt.Commentf("service should be not nil"))
}

func TestService_ListRepos(t *testing.T) {
	c := qt.New(t)
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

	err := svc.ListRepos(ctx)
	c.Assert(err, qt.IsNil)
	c.Assert(store.FindReposInvoked, qt.IsTrue, qt.Commentf("FindRepos() should be called"))
}
