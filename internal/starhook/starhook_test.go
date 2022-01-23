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
	svc := NewService(nil, nil, nil)
	c.Assert(svc, qt.Not(qt.IsNil), qt.Commentf("service should be not nil"))
}

func TestService_ListRepos(t *testing.T) {
	c := qt.New(t)
	ctx := context.Background()
	repos := []*internal.Repository{
		{ID: 1},
		{ID: 2},
	}

	store := &mock.MetadataStore{
		FindReposFn: func(ctx context.Context, filter internal.RepositoryFilter, opt internal.FindOptions) ([]*internal.Repository, error) {
			return repos, nil
		},
	}

	fsstore := &mock.RepositoryStore{}

	svc := NewService(nil, store, fsstore)

	resp, err := svc.ListRepos(ctx)

	c.Assert(err, qt.IsNil)
	c.Assert(resp, qt.DeepEquals, repos)
	c.Assert(store.FindReposInvoked, qt.IsTrue, qt.Commentf("FindRepos() should be called"))
}
