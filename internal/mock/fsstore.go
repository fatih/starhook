package mock

import (
	"context"

	"github.com/fatih/starhook/internal"
)

var _ internal.RepositoryStore = (*RepositoryStore)(nil)

// RepositoryStore represents a mock implementation of internal.RepositoryStore.
type RepositoryStore struct {
	CreateRepoFn      func(ctx context.Context, repo *internal.Repository) error
	CreateRepoInvoked bool

	UpdateRepoFn      func(ctx context.Context, repo *internal.Repository) error
	UpdateRepoInvoked bool

	DeleteRepoFn      func(ctx context.Context, repo *internal.Repository) error
	DeleteRepoInvoked bool
}

// CreateRepository creates a single repository and returns the ID.
func (r *RepositoryStore) CreateRepo(ctx context.Context, repo *internal.Repository) error {
	r.CreateRepoInvoked = true
	return r.CreateRepoFn(ctx, repo)
}

// UpdateRepo updates a single repository
func (r *RepositoryStore) UpdateRepo(ctx context.Context, repo *internal.Repository) error {
	r.UpdateRepoInvoked = true
	return r.UpdateRepoFn(ctx, repo)
}

// DeleteRepo deletes a single repository
func (r *RepositoryStore) DeleteRepo(ctx context.Context, repo *internal.Repository) error {
	r.DeleteRepoInvoked = true
	return r.DeleteRepoFn(ctx, repo)
}
