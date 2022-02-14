package fsstore

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/fatih/starhook/internal"
	"github.com/fatih/starhook/internal/git"
)

type RepositoryStore struct {
	dir string
}

func NewRepositoryStore(dir string) (*RepositoryStore, error) {
	return &RepositoryStore{
		dir: dir,
	}, nil
}

// CreateRepos creates a single repository.
func (r *RepositoryStore) CreateRepo(ctx context.Context, repo *internal.Repository) error {
	repoDir := filepath.Join(r.dir, repo.Name)

	// do not clone if it exists
	if _, err := os.Stat(repoDir); err == nil {
		return nil
	}

	log.Printf("[DEBUG]  cloning repo, owner: %q, name: %q, branch: %q",
		repo.Owner, repo.Name, repo.Branch)
	cloneURL := fmt.Sprintf("https://github.com/%s/%s.git", repo.Owner, repo.Name)
	g := &git.Client{}
	_, err := g.Run("clone", cloneURL, "--depth=1", repoDir)
	if err != nil {
		return err
	}

	return nil
}

// CreateRepos updates a single repository.
func (r *RepositoryStore) UpdateRepo(ctx context.Context, repo *internal.Repository) error {
	repoDir := filepath.Join(r.dir, repo.Name)
	g := &git.Client{Dir: repoDir}

	log.Printf("[DEBUG] updating repo, owner: %q, name: %q, branch: %q",
		repo.Owner, repo.Name, repo.Branch)

	if _, err := g.Run("reset", "--hard"); err != nil {
		return err
	}
	if _, err := g.Run("clean", "-df"); err != nil {
		return err
	}
	if _, err := g.Run("checkout", repo.Branch); err != nil {
		return err
	}
	if _, err := g.Run("pull", "origin", repo.SHA); err != nil {
		return err
	}

	return nil
}

// DeleteRepo deletes a single repository.
func (r *RepositoryStore) DeleteRepo(ctx context.Context, repo *internal.Repository) error {
	log.Printf("[DEBUG]  deleting repo, owner: %q, name: %q, branch: %q",
		repo.Owner, repo.Name, repo.Branch)

	repoDir := filepath.Join(r.dir, repo.Name)
	return os.RemoveAll(repoDir)
}
