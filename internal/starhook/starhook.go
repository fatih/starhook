package starhook

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"sync"
	"time"

	"github.com/fatih/starhook/internal"
	"github.com/fatih/starhook/internal/gh"
	"github.com/hashicorp/go-multierror"

	"golang.org/x/sync/semaphore"
)

type Service struct {
	client *gh.Client
	store  internal.MetadataStore
	fs     internal.RepositoryStore
}

func NewService(ghClient *gh.Client, store internal.MetadataStore, fs internal.RepositoryStore) *Service {
	return &Service{
		client: ghClient,
		store:  store,
		fs:     fs,
	}
}

// DeleteRepo deletes the given repo from the DB and the folder if it's exist.
func (s *Service) DeleteRepo(ctx context.Context, repoID int64) error {
	repo, err := s.store.FindRepo(ctx, repoID)
	if err != nil {
		return err
	}

	err = s.store.DeleteRepo(ctx, internal.RepositoryBy{RepoID: &repoID})
	if err != nil {
		return err
	}

	err = s.fs.DeleteRepo(ctx, repo)
	if err != nil {
		return err
	}

	return nil
}

// ListRepos lists all the repositories.
func (s *Service) ListRepos(ctx context.Context) ([]*internal.Repository, error) {
	return s.store.FindRepos(ctx, internal.RepositoryFilter{}, internal.DefaultFindOptions)
}

// ReposToUpdate returns the repositories to clone or update.
func (s *Service) ReposToUpdate(ctx context.Context) ([]*internal.Repository, []*internal.Repository, error) {
	repos, err := s.ListRepos(ctx)
	if err != nil {
		return nil, nil, err
	}

	if len(repos) == 0 {
		return nil, nil, errors.New("no repositories to update")
	}

	var (
		clone  []*internal.Repository
		update []*internal.Repository
	)

	for _, repo := range repos {
		if repo.SyncedAt.IsZero() {
			log.Printf("[DEBUG] clone, owner: %q, name: %q, branch: %q, sha: %q",
				repo.Owner, repo.Name, repo.Branch, repo.SHA)
			clone = append(clone, repo)
			continue
		}

		if repo.SyncedAt.Before(repo.BranchUpdatedAt) {
			log.Printf("[DEBUG] update, owner: %q, name: %q, branch: %q, sha: %q",
				repo.Owner, repo.Name, repo.Branch, repo.SHA)
			update = append(update, repo)
		}

	}

	return clone, update, nil
}

// CloneRepos clones the given repositories.
func (s *Service) CloneRepos(ctx context.Context, repos []*internal.Repository) error {
	if len(repos) == 0 {
		return nil
	}

	var wg sync.WaitGroup
	var errs *multierror.Error
	var mu sync.Mutex

	const maxWorkers = 10
	sem := semaphore.NewWeighted(maxWorkers)

	for _, repo := range repos {
		wg.Add(1)
		repo := repo

		err := sem.Acquire(ctx, 1)
		if err != nil {
			return fmt.Errorf("couldn't acquire semaphore: %s", err)
		}

		go func() {
			defer sem.Release(1)
			defer wg.Done()

			if err := s.cloneRepo(ctx, repo); err != nil {
				mu.Lock()
				errs = multierror.Append(errs, err)
				mu.Unlock()
			}
		}()
	}

	wg.Wait()

	return errs.ErrorOrNil()
}

// cloneRepo clones a single repository.
func (s *Service) cloneRepo(ctx context.Context, repo *internal.Repository) error {
	err := s.fs.CreateRepo(ctx, repo)
	if err != nil {
		return err
	}

	now := time.Now().UTC()
	err = s.store.UpdateRepo(ctx,
		internal.RepositoryBy{
			Name: &repo.Name,
		},
		internal.RepositoryUpdate{
			SyncedAt: &now,
		},
	)

	return err
}

// updateRepo updates a single repository.
func (s *Service) updateRepo(ctx context.Context, repo *internal.Repository) error {
	err := s.fs.UpdateRepo(ctx, internal.UpdateOptions{}, repo)
	if os.IsNotExist(err) {
		// this happens if the folder was deleted not with starhook. Remove it
		// from the repository store and repair any incosistency
		log.Printf("[DEBUG] repository was removed from file system, removing from metadastore owner: %q, name: %q, branch: %q",
			repo.Owner, repo.Name, repo.Branch)

		return s.store.DeleteRepo(ctx, internal.RepositoryBy{RepoID: &repo.ID})
	}

	if err != nil {
		return err
	}

	now := time.Now().UTC()
	err = s.store.UpdateRepo(ctx,
		internal.RepositoryBy{
			Name: &repo.Name,
		},
		internal.RepositoryUpdate{
			SyncedAt: &now,
		},
	)

	return err
}

// UpdateRepos updates the given repositories locally to its latest ref.
func (s *Service) UpdateRepos(ctx context.Context, repos []*internal.Repository) error {
	if len(repos) == 0 {
		return nil
	}

	var wg sync.WaitGroup
	var errs *multierror.Error
	var mu sync.Mutex

	const maxWorkers = 10
	sem := semaphore.NewWeighted(maxWorkers)

	for _, repo := range repos {
		wg.Add(1)
		repo := repo

		err := sem.Acquire(ctx, 1)
		if err != nil {
			return fmt.Errorf("couldn't acquire semaphore: %s", err)
		}

		go func() {
			defer sem.Release(1)
			defer wg.Done()

			if err := s.updateRepo(ctx, repo); err != nil {
				mu.Lock()
				errs = multierror.Append(errs, err)
				mu.Unlock()
			}
		}()

	}

	wg.Wait()

	return errs.ErrorOrNil()
}

// SyncRepos syncs the repositories in the store, with the fetched remote repositories.
func (s *Service) SyncRepos(ctx context.Context, repos, fetchedRepos []*internal.Repository) error {
	localRepos := make(map[string]*internal.Repository, len(repos))
	for _, repo := range repos {
		localRepos[repo.Nwo] = repo
	}

	var wg sync.WaitGroup
	var errs *multierror.Error
	var mu sync.Mutex

	const maxWorkers = 5
	sem := semaphore.NewWeighted(maxWorkers)

	log.Printf("[DEBUG] syncing with local store, fetched repos: %d local repos: %d", len(fetchedRepos), len(localRepos))
	for _, repo := range fetchedRepos {
		wg.Add(1)
		repo := repo
		localRepo := localRepos[repo.Nwo]

		if err := sem.Acquire(ctx, 1); err != nil {
			return fmt.Errorf("couldn't acquire semaphore: %s", err)
		}

		go func() {
			defer sem.Release(1)
			defer wg.Done()

			if err := s.syncRepo(ctx, localRepo, repo); err != nil {
				log.Printf("[ERROR] retrieving branch information, owner: %q, name: %q, branch: %q, err: %s",
					repo.Owner, repo.Name, repo.Branch, err)
				mu.Lock()
				errs = multierror.Append(errs, err)
				mu.Unlock()
			}
		}()
	}

	wg.Wait()

	return errs.ErrorOrNil()
}

// syncRepo sync the local repo with the fetched repo
func (s *Service) syncRepo(ctx context.Context, localRepo, repo *internal.Repository) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	// NOTE(fatih): there is the possibility that the default branch
	// might have changed, for now we assume that's not the case, but
	// it's worth noting here.
	branch, err := s.client.Branch(ctx, repo.Owner, repo.Name, repo.Branch)
	if err != nil {
		if errors.Is(err, gh.ErrBranchNotFound) {
			log.Printf("[DEBUG] no branch information found, owner: %q, name: %q, branch: %q",
				repo.Owner, repo.Name, repo.Branch)
			return nil
		}

		return err
	}

	repo.BranchUpdatedAt = branch.UpdatedAt
	repo.SHA = branch.SHA

	if localRepo == nil {
		log.Printf("[DEBUG] creating new entry, owner: %q, name: %q, branch: %q",
			repo.Owner, repo.Name, repo.Branch)
		_, err = s.store.CreateRepo(ctx, repo)
		if err != nil {
			return err
		}
	} else if !localRepo.BranchUpdatedAt.Equal(repo.BranchUpdatedAt) {
		log.Printf("[DEBUG] updating entry, owner: %q, name: %q, branch: %q",
			repo.Owner, repo.Name, repo.Branch)
		err = s.store.UpdateRepo(ctx,
			internal.RepositoryBy{
				Name: &repo.Name,
			},
			internal.RepositoryUpdate{
				SHA:             &repo.SHA,
				BranchUpdatedAt: &repo.BranchUpdatedAt,
			},
		)
		if err != nil {
			return err
		}
	}

	return nil
}
