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
	ID              int64
	Nwo             string // name with owner
	Owner           string // i.e: fatih, github
	Name            string // i.e: vim-go, gh-ost
	Branch          string // usually it's master, but people can change it
	BranchUpdatedAt time.Time

	CreatedAt time.Time // time this object was created to the store
	UpdatedAt time.Time // time this object was updated in the store
}

type RepositoryFilter struct{}

type RepositoryUpdate struct {
	Nwo   *string
	Owner *string
}

type RepositoryBy struct {
	RepoID *int64
	Name   *string
}

type RepositoryService interface {
	// FetchRepos fetches all repositories for the given query
	FetchRepos(ctx context.Context, query string) ([]*Repository, error)

	// UpdateRepos updates all the repositories for the given query
	UpdateRepos(ctx context.Context, query string) ([]*Repository, error)
}

type RepositoryStore interface {
	// FindRepositories returns a list of repositories
	FindRepos(ctx context.Context, filter RepositoryFilter, opt FindOptions) ([]*Repository, error)

	// FindRepo returns the *Repository with the given ID
	FindRepo(ctx context.Context, repoID int64) (*Repository, error)

	// CreateRepository creates a single repository and returns the ID.
	CreateRepo(context.Context, *Repository) (int64, error)

	// UpdateRepo updates a single repository
	UpdateRepo(ctx context.Context, by RepositoryBy, upd RepositoryUpdate) error
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
