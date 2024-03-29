package starhook

import (
	"context"
	"errors"
	"log"
	"os"
	"time"

	"github.com/fatih/starhook/internal"
	"github.com/fatih/starhook/internal/gh"

	"github.com/fatih/semgroup"
)

type Service struct {
	client *gh.Client
	store  internal.MetadataStore
	fs     internal.RepositoryStore
}

type SyncRepos struct {
	Clone  []*internal.Repository
	Update []*internal.Repository
	Delete []*internal.Repository
}

func NewService(ghClient *gh.Client, store internal.MetadataStore, fs internal.RepositoryStore) *Service {
	return &Service{
		client: ghClient,
		store:  store,
		fs:     fs,
	}
}

// ListRepos lists all the repositories.
func (s *Service) ListRepos(ctx context.Context) ([]*internal.Repository, error) {
	return s.store.FindRepos(ctx, internal.RepositoryFilter{}, internal.DefaultFindOptions)
}

// DeleteRepos deletes the given repositories.
func (s *Service) DeleteRepos(ctx context.Context, repos []*internal.Repository) error {
	if len(repos) == 0 {
		return nil
	}

	const maxWorkers = 10
	sem := semgroup.NewGroup(ctx, maxWorkers)

	for _, repo := range repos {
		repo := repo

		sem.Go(func() error {
			return s.deleteRepo(ctx, repo)
		})
	}

	return sem.Wait()
}

// deleteRepo deletes the given repo from the DB and the folder if it's exist.
func (s *Service) deleteRepo(ctx context.Context, repo *internal.Repository) error {
	err := s.fs.DeleteRepo(ctx, repo)
	if err != nil {
		return err
	}

	repoID := repo.ID
	return s.store.DeleteRepo(ctx, internal.RepositoryBy{RepoID: &repoID})
}

// CloneRepos clones the given repositories.
func (s *Service) CloneRepos(ctx context.Context, repos []*internal.Repository) error {
	if len(repos) == 0 {
		return nil
	}

	const maxWorkers = 10
	sem := semgroup.NewGroup(ctx, maxWorkers)

	for _, repo := range repos {
		repo := repo

		sem.Go(func() error {
			return s.cloneRepo(ctx, repo)
		})
	}

	return sem.Wait()
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

	const maxWorkers = 10
	sem := semgroup.NewGroup(ctx, maxWorkers)

	for _, repo := range repos {
		repo := repo

		sem.Go(func() error {
			return s.updateRepo(ctx, repo)
		})
	}

	return sem.Wait()
}

// SyncRepos syncs the repositories in the store, with the fetched remote
// repositories and returns the repositories to clone, update or delete.
func (s *Service) SyncRepos(ctx context.Context, repos, fetched []*internal.Repository) (*SyncRepos, error) {
	var (
		clone   []*internal.Repository
		update  []*internal.Repository
		deleted []*internal.Repository
	)

	localRepos := make(map[string]*internal.Repository, len(repos))
	for _, repo := range repos {
		localRepos[repo.Nwo] = repo
	}

	fetchedRepos := make(map[string]*internal.Repository, len(fetched))
	for _, repo := range fetched {
		fetchedRepos[repo.Nwo] = repo
	}

	const maxWorkers = 5
	sem := semgroup.NewGroup(ctx, maxWorkers)

	// check for repos to delete
	for _, repo := range localRepos {
		if _, ok := fetchedRepos[repo.Nwo]; !ok {
			// local repository doesn't exist in the final, fetched list, needs
			// to be removed
			deleted = append(deleted, repo)
			delete(localRepos, repo.Nwo)
		}
	}

	log.Printf("[DEBUG] syncing with local store, fetched repos: %d local repos: %d", len(fetchedRepos), len(localRepos))
	// check for repos to update or clone
	for _, repo := range fetchedRepos {
		repo := repo
		localRepo := localRepos[repo.Nwo]

		sem.Go(func() error {
			return s.syncRepo(ctx, localRepo, repo)
		})
	}

	if err := sem.Wait(); err != nil {
		return nil, err
	}

	syncedRepos := make([]*internal.Repository, 0)

	// TODO(fatih): use a more efficient fetching, dont do it one by one
	for _, repo := range localRepos {
		rp, err := s.store.FindRepo(ctx, repo.ID)
		if err != nil {
			return nil, err
		}

		syncedRepos = append(syncedRepos, rp)
	}

	for _, repo := range syncedRepos {
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

	return &SyncRepos{
		Clone:  clone,
		Update: update,
		Delete: deleted,
	}, nil
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
