package internal

import (
	"context"
	"errors"
	"fmt"
	"time"
)

var ErrNotFound = errors.New("not found")

// Repository represents a repository on GitHub
type Repository struct {
	ID     int64
	Nwo    string // name with owner
	Owner  string // i.e: fatih, github
	Name   string // i.e: vim-go, gh-ost
	Branch string // usually it's main, but people can change it
	SHA    string // commit SHA, saved during sync

	// SyncedAt defines the time the repo content was synced locally. If
	// BrancUpdatedAt > SyncedAt, it means the localy copy is out of date and
	// needs to be updated. A zero time means it's not synced (cloned) locally.
	SyncedAt time.Time

	// BranchUpdatedAt defines the time the branch was updated on GitHub
	BranchUpdatedAt time.Time

	CreatedAt time.Time // time this object was created in the store
	UpdatedAt time.Time // time this object was updated in the store
}

type RepositoryFilter struct{}

// RepositoryUpdate is used to update a Repository
type RepositoryUpdate struct {
	Nwo             *string
	Owner           *string
	SHA             *string
	SyncedAt        *time.Time
	BranchUpdatedAt *time.Time
}

// RepositoryUpdate is used to select a repository to update
type RepositoryBy struct {
	RepoID *int64
	Name   *string
}

// MetadataStore manages the information about repositories.
type MetadataStore interface {
	// FindRepositories returns a list of repositories
	FindRepos(ctx context.Context, filter RepositoryFilter, opt FindOptions) ([]*Repository, error)

	// FindRepo returns the *Repository with the given ID
	FindRepo(ctx context.Context, repoID int64) (*Repository, error)

	// CreateRepository creates a single repository and returns the ID.
	CreateRepo(ctx context.Context, repo *Repository) (int64, error)

	// UpdateRepo updates a single repository
	UpdateRepo(ctx context.Context, by RepositoryBy, upd RepositoryUpdate) error

	// DeleteRepo deletes a single repository
	DeleteRepo(ctx context.Context, by RepositoryBy) error
}

// RepositoryStore manages the repositories on a filesystem.
type RepositoryStore interface {
	// CreateRepos creates a single repository.
	CreateRepo(ctx context.Context, repo *Repository) error

	// CreateRepos updates a single repository.
	UpdateRepo(ctx context.Context, repo *Repository) error

	// DeleteRepo deletes a single repository.
	DeleteRepo(ctx context.Context, repo *Repository) error
}

// DefaultFindOptions is the default option to be used with Find* methods
var DefaultFindOptions = FindOptions{
	Limit: 25,
}

// FindOptions is passed to methods who require to specifcy how to find their
// resources.
type FindOptions struct {
	Offset     int
	Limit      int
	SortBy     string
	Descending bool
}

// SortByDirection returns the sort directive for a given resource.
func (f FindOptions) SortByDirection() string {
	if f.Descending {
		return fmt.Sprintf("%s desc", f.SortBy)
	}
	return fmt.Sprintf("%s asc", f.SortBy)
}
