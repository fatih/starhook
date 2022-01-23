package starhook

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/fatih/starhook/internal"
	"github.com/fatih/starhook/internal/gh"

	"golang.org/x/sync/errgroup"
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
func (s *Service) ReposToUpdate(ctx context.Context, repos []*internal.Repository) ([]*internal.Repository, []*internal.Repository, error) {
	if len(repos) == 0 {
		return nil, nil, errors.New("no repositories to update")
	}

	var (
		clone  []*internal.Repository
		update []*internal.Repository
	)

	for _, repo := range repos {
		if repo.SyncedAt.IsZero() {
			clone = append(clone, repo)
			continue
		}

		if repo.SyncedAt.Before(repo.BranchUpdatedAt) {
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

	const maxWorkers = 10
	sem := semaphore.NewWeighted(maxWorkers)

	g, ctx := errgroup.WithContext(ctx)
	for _, repo := range repos {
		repo := repo

		err := sem.Acquire(ctx, 1)
		if err != nil {
			return fmt.Errorf("couldn't acquire semaphore: %s", err)
		}

		g.Go(func() error {
			defer sem.Release(1)
			return s.fs.CreateRepo(ctx, repo)
		})
	}

	if err := g.Wait(); err != nil {
		return fmt.Errorf("errgroup wait err: %s ", err)
	}

	for _, repo := range repos {
		now := time.Now().UTC()
		err := s.store.UpdateRepo(ctx,
			internal.RepositoryBy{
				Name: &repo.Name,
			},
			internal.RepositoryUpdate{
				SyncedAt: &now,
			},
		)
		if err != nil {
			return err
		}
	}

	return nil
}

// UpdateRepos updates the given repositories locally to its latest ref.
func (s *Service) UpdateRepos(ctx context.Context, repos []*internal.Repository) error {
	if len(repos) == 0 {
		return nil
	}

	const maxWorkers = 10
	sem := semaphore.NewWeighted(maxWorkers)

	g, ctx := errgroup.WithContext(ctx)
	for _, repo := range repos {
		repo := repo

		err := sem.Acquire(ctx, 1)
		if err != nil {
			return fmt.Errorf("couldn't acquire semaphore: %s", err)
		}

		g.Go(func() error {
			defer sem.Release(1)
			return s.fs.UpdateRepo(ctx, repo)
		})
	}

	if err := g.Wait(); err != nil {
		return fmt.Errorf("errgroup wait err: %s ", err)
	}

	for _, repo := range repos {
		now := time.Now().UTC()
		err := s.store.UpdateRepo(ctx,
			internal.RepositoryBy{
				Name: &repo.Name,
			},
			internal.RepositoryUpdate{
				SyncedAt: &now,
			},
		)
		if err != nil {
			return err
		}
	}

	return nil
}

// SyncRepos syncs the repositories in the store, with the fetched remote repositories.
func (s *Service) SyncRepos(ctx context.Context, repos, fetchedRepos []*internal.Repository) error {
	localRepos := make(map[string]*internal.Repository, len(repos))
	for _, repo := range repos {
		localRepos[repo.Nwo] = repo
	}

	g, ctx := errgroup.WithContext(ctx)

	const maxWorkers = 5
	sem := semaphore.NewWeighted(maxWorkers)

	for _, repo := range fetchedRepos {
		repo := repo

		if err := sem.Acquire(ctx, 1); err != nil {
			return fmt.Errorf("couldn't acquire semaphore: %s", err)
		}

		g.Go(func() error {
			defer sem.Release(1)

			// NOTE(fatih): there is the possibility that the default branch
			// might have changed, for now we assume that's not the case, but
			// it's worth noting here.
			branch, err := s.client.Branch(ctx, repo.Owner, repo.Name, repo.Branch)
			if err != nil {
				return err
			}
			repo.BranchUpdatedAt = branch.UpdatedAt
			repo.SHA = branch.SHA

			localRepo, ok := localRepos[repo.Nwo]
			if !ok {
				_, err = s.store.CreateRepo(ctx, repo)
				if err != nil {
					return err
				}
			} else if !localRepo.BranchUpdatedAt.Equal(repo.BranchUpdatedAt) {
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
		})
	}

	return g.Wait()
}
