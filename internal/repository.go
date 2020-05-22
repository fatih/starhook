package internal

import (
	"context"
	"time"
)

// Repository represents a repository on GitHub
type Repository struct {
	ID        int64
	Name      string // i.e: vim-go, gh-ost
	Owner     string // i.e: fatih, github
	CreatedAt time.Time
	UpdatedAt time.Time
}

type RepositoryUpdate struct{}

type RepositoryService interface {
	// FetchRepos fetches all repositories
	FetchRepos(ctx context.Context) ([]*Repository, error)

	// UpdateRepos
	UpdateRepos(ctx context.Context) ([]*Repository, error)
}

type RepositoryStore interface {
	// FindRepositories returns a list of repositories
	FindRepos(ctx context.Context) ([]*Repository, error)

	// CreateRepository creates a single repository and returns the ID.
	CreateRepo(context.Context, *Repository) (int64, error)

	// UpdateRepo creates a single repository
	UpdateRepo(ctx context.Context, repoID int64, upd RepositoryUpdate) error
}
