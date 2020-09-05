package starhook

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/dustin/go-humanize"
	"github.com/fatih/starhook/internal"
	"github.com/fatih/starhook/internal/gh"
	"github.com/fatih/starhook/internal/git"

	"golang.org/x/sync/errgroup"
	"golang.org/x/sync/semaphore"
)

type result struct {
	updated bool
	created bool
}

type Service struct {
	client *gh.Client
	dir    string
	update bool
	store  internal.RepositoryStore
}

func NewService(ghClient *gh.Client, store internal.RepositoryStore, dir string) *Service {
	return &Service{
		client: ghClient,
		dir:    dir,
		store:  store,
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

	err = s.deleteRepo(ctx, repo)
	if err != nil {
		return err
	}

	fmt.Printf("==> removed repository: %q\n", repo.Nwo)
	return nil
}

// ListRepos lists all the repositories.
func (s *Service) ListRepos(ctx context.Context) error {
	repos, err := s.store.FindRepos(ctx, internal.RepositoryFilter{}, internal.DefaultFindOptions)
	if err != nil {
		return err
	}

	lastUpdated := time.Time{}
	for _, repo := range repos {
		if repo.UpdatedAt.After(lastUpdated) {
			lastUpdated = repo.UpdatedAt
		}
		fmt.Printf("%3d %s\n", repo.ID, repo.Nwo)
	}

	fmt.Printf("==> local %d repositories (last synced: %s)\n", len(repos), humanize.Time(lastUpdated))
	return nil
}

// ReposToUpdate returns the repositories to clone or update.
func (s *Service) ReposToUpdate(ctx context.Context) ([]*internal.Repository, []*internal.Repository, error) {
	repos, err := s.store.FindRepos(ctx, internal.RepositoryFilter{}, internal.DefaultFindOptions)
	if err != nil {
		return nil, nil, err
	}

	if len(repos) == 0 {
		return nil, nil, errors.New("no repositories to update, please sync first")
	}

	lastUpdated := time.Time{}
	for _, repo := range repos {
		if repo.UpdatedAt.After(lastUpdated) {
			lastUpdated = repo.UpdatedAt
		}
	}

	fmt.Printf("==> have %d repositories. last synced: %s\n", len(repos), humanize.Time(lastUpdated))

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
	if err := s.cloneRepos(ctx, repos); err != nil {
		return err
	}

	for _, repo := range repos {
		fmt.Printf("  %q is created\n", repo.Name)
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
	if err := s.updateRepos(ctx, repos); err != nil {
		return err
	}

	for _, repo := range repos {
		fmt.Printf("  %q is updated (last updated: %s)\n",
			repo.Name, humanize.Time(repo.SyncedAt))

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

// SyncRepos syncs the remote repositories metadata with the store data.
func (s *Service) SyncRepos(ctx context.Context, fetchedRepos []*internal.Repository) error {

	repos, err := s.store.FindRepos(ctx, internal.RepositoryFilter{}, internal.DefaultFindOptions)
	if err != nil {
		return err
	}

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
			fmt.Printf("acquire err = %+v\n", err)
			break
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

func (s *Service) updateRepos(ctx context.Context, repos []*internal.Repository) error {
	if len(repos) == 0 {
		return nil
	}
	fmt.Println("==> updating repositories")
	start := time.Now()

	const maxWorkers = 10
	sem := semaphore.NewWeighted(maxWorkers)

	g, ctx := errgroup.WithContext(ctx)
	for _, repo := range repos {
		repo := repo

		err := sem.Acquire(ctx, 1)
		if err != nil {
			fmt.Printf("acquire err = %+v\n", err)
			break
		}

		g.Go(func() error {
			defer sem.Release(1)
			return s.updateRepo(ctx, repo)
		})
	}

	if err := g.Wait(); err != nil {
		fmt.Printf("g.Wait() err = %+v\n", err)
	}

	fmt.Printf("==> updated: %d repositories (elapsed time: %s)\n",
		len(repos), time.Since(start).String())
	return nil
}

func (s *Service) updateRepo(ctx context.Context, repo *internal.Repository) error {
	fmt.Printf("  updating %s\n", repo.Name)
	repoDir := filepath.Join(s.dir, repo.Name)
	g := &git.Client{Dir: repoDir}

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

func (s *Service) cloneRepos(ctx context.Context, repos []*internal.Repository) error {
	if len(repos) == 0 {
		return nil
	}

	fmt.Println("==> cloning repositories")
	start := time.Now()

	const maxWorkers = 10
	sem := semaphore.NewWeighted(maxWorkers)

	g, ctx := errgroup.WithContext(ctx)
	for _, repo := range repos {
		repo := repo

		err := sem.Acquire(ctx, 1)
		if err != nil {
			fmt.Printf("acquire err = %+v\n", err)
			break
		}

		g.Go(func() error {
			defer sem.Release(1)
			return s.cloneRepo(ctx, repo)
		})
	}

	if err := g.Wait(); err != nil {
		fmt.Printf("g.Wait() err = %+v\n", err)
	}

	fmt.Printf("==> cloned: %d repositories (elapsed time: %s)\n",
		len(repos), time.Since(start).String())
	return nil
}

func (s *Service) cloneRepo(ctx context.Context, repo *internal.Repository) error {
	repoDir := filepath.Join(s.dir, repo.Name)

	// do not clone if it exists
	if _, err := os.Stat(repoDir); err == nil {
		return nil
	}

	cloneURL := fmt.Sprintf("https://github.com/%s/%s.git", repo.Owner, repo.Name)
	fmt.Printf("  cloning %s\n", repo.Name)
	g := &git.Client{}
	_, err := g.Run("clone", cloneURL, "--depth=1", repoDir)
	if err != nil {
		return err
	}

	return nil
}

func (s *Service) deleteRepo(ctx context.Context, repo *internal.Repository) error {
	repoDir := filepath.Join(s.dir, repo.Name)
	return os.RemoveAll(repoDir)
}
