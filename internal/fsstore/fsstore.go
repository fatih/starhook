package fsstore

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

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

// CreateRepo creates a single repository.
func (r *RepositoryStore) CreateRepo(ctx context.Context, repo *internal.Repository) error {
	repoDir := filepath.Join(r.dir, repo.Name)

	// do not clone if it exists
	if _, err := os.Stat(repoDir); err == nil {
		return nil
	}

	log.Printf("[DEBUG] cloning repo, owner: %q, name: %q, branch: %q",
		repo.Owner, repo.Name, repo.Branch)

	g := &git.Client{}

	cloneURL := fmt.Sprintf("https://github.com/%s/%s.git", repo.Owner, repo.Name)
	_, err := g.Run("clone", cloneURL, "--depth=1", repoDir)
	if err != nil {
		return err
	}

	return nil
}

// UpdateRepo updates a single repository.
func (r *RepositoryStore) UpdateRepo(ctx context.Context, opts internal.UpdateOptions, repo *internal.Repository) error {
	repoDir := filepath.Join(r.dir, repo.Name)

	// don't continue if the repo was removed or doesn't exist.
	_, err := os.Stat(repoDir)
	if err != nil {
		return err
	}

	g := &git.Client{Dir: repoDir}

	log.Printf("[DEBUG] updating repo, name: %q, branch: %q, sha: %q (opts: %v)",
		repo.Nwo, repo.Branch, repo.SHA, opts)

	if opts.ForceClean {
		// this option assumes a immutable set of repositories that always
		// track the latest
		if _, err := g.Run("reset", "--hard"); err != nil {
			return err
		}
		if _, err := g.Run("clean", "-df"); err != nil {
			return err
		}
		if _, err := g.Run("checkout", repo.Branch); err != nil {
			return err
		}
		if _, err := g.Run("pull", "--rebase", "origin", repo.SHA); err != nil {
			return err
		}
	} else {
		branch, err := g.Run("rev-parse", "--abbrev-ref", "HEAD")
		if err != nil {
			return err
		}

		// TODO(fatih) make sure remote name is indeed 'origin'.
		if string(branch) == repo.Branch {
			if _, err := g.Run("pull", "--rebase", "origin", repo.SHA); err != nil {
				return err
			}
		} else {
			out, err := g.Run("stash", "push")
			if err != nil {
				return err
			}

			popNeeded := true
			if strings.Contains(string(out), "No local changes to save") {
				popNeeded = false
			}

			if _, err := g.Run("checkout", repo.Branch); err != nil {
				return err
			}

			// NOTE(fatih): check whether git reset would be better for our usecase.
			// or better: "git reset --hard origin/main"
			if _, err := g.Run("pull", "--rebase", "origin", repo.SHA); err != nil {
				return err
			}

			if _, err := g.Run("checkout", "-"); err != nil {
				return err
			}

			if popNeeded {
				if _, err := g.Run("stash", "pop"); err != nil {
					return err
				}
			}
		}
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
