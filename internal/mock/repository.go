package mock

import (
	"context"

	"github.com/fatih/starhook/internal"
)

var _ internal.MetadataStore = (*MetadataStore)(nil)

// MetadataStore represents a mock implementation of internal.MetadataStore.
type MetadataStore struct {
	FindReposFn      func(ctx context.Context, filter internal.RepositoryFilter, opt internal.FindOptions) ([]*internal.Repository, error)
	FindReposInvoked bool

	FindRepoFn      func(ctx context.Context, repoID int64) (*internal.Repository, error)
	FindRepoInvoked bool

	CreateRepoFn      func(ctx context.Context, repo *internal.Repository) (int64, error)
	CreateRepoInvoked bool

	UpdateRepoFn      func(ctx context.Context, by internal.RepositoryBy, upd internal.RepositoryUpdate) error
	UpdateRepoInvoked bool

	DeleteRepoFn      func(ctx context.Context, by internal.RepositoryBy) error
	DeleteRepoInvoked bool
}

// FindRepositories returns a list of repositories
func (r *MetadataStore) FindRepos(ctx context.Context, filter internal.RepositoryFilter, opt internal.FindOptions) ([]*internal.Repository, error) {
	r.FindReposInvoked = true
	return r.FindReposFn(ctx, filter, opt)
}

// FindRepo returns the *Repository with the given ID
func (r *MetadataStore) FindRepo(ctx context.Context, repoID int64) (*internal.Repository, error) {
	r.FindRepoInvoked = true
	return r.FindRepoFn(ctx, repoID)
}

// CreateRepository creates a single repository and returns the ID.
func (r *MetadataStore) CreateRepo(ctx context.Context, repo *internal.Repository) (int64, error) {
	r.CreateRepoInvoked = true
	return r.CreateRepoFn(ctx, repo)
}

// UpdateRepo updates a single repository
func (r *MetadataStore) UpdateRepo(ctx context.Context, by internal.RepositoryBy, upd internal.RepositoryUpdate) error {
	r.UpdateRepoInvoked = true
	return r.UpdateRepoFn(ctx, by, upd)
}

// DeleteRepo deletes a single repository
func (r *MetadataStore) DeleteRepo(ctx context.Context, by internal.RepositoryBy) error {
	r.DeleteRepoInvoked = true
	return r.DeleteRepoFn(ctx, by)
}
