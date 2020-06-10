package mock

import (
	"context"

	"github.com/fatih/starhook/internal"
)

var _ internal.RepositoryStore = (*RepositoryStore)(nil)

// RepositoryStore represents a mock implementation of internal.RepositoryStore.
type RepositoryStore struct {
	FindReposFn      func(ctx context.Context, filter internal.RepositoryFilter, opt internal.FindOptions) ([]*internal.Repository, error)
	FindReposInvoked bool

	FindRepoFn      func(ctx context.Context, repoID int64) (*internal.Repository, error)
	FindRepoInvoked bool

	CreateRepoFn      func(ctx context.Context, repo *internal.Repository) (int64, error)
	CreateRepoInvoked bool

	UpdateRepoFn      func(ctx context.Context, by internal.RepositoryBy, upd internal.RepositoryUpdate) error
	UpdateRepoInvoked bool
}

// FindRepositories returns a list of repositories
func (r *RepositoryStore) FindRepos(ctx context.Context, filter internal.RepositoryFilter, opt internal.FindOptions) ([]*internal.Repository, error) {
	r.FindReposInvoked = true
	return r.FindReposFn(ctx, filter, opt)
}

// FindRepo returns the *Repository with the given ID
func (r *RepositoryStore) FindRepo(ctx context.Context, repoID int64) (*internal.Repository, error) {
	r.FindRepoInvoked = true
	return r.FindRepoFn(ctx, repoID)
}

// CreateRepository creates a single repository and returns the ID.
func (r *RepositoryStore) CreateRepo(ctx context.Context, repo *internal.Repository) (int64, error) {
	r.CreateRepoInvoked = true
	return r.CreateRepoFn(ctx, repo)
}

// UpdateRepo updates a single repository
func (r *RepositoryStore) UpdateRepo(ctx context.Context, by internal.RepositoryBy, upd internal.RepositoryUpdate) error {
	r.UpdateRepoInvoked = true
	return r.UpdateRepoFn(ctx, by, upd)
}
