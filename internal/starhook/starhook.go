package starhook

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/dustin/go-humanize"
	"github.com/fatih/starhook/internal"
	"github.com/fatih/starhook/internal/gh"
	"github.com/fatih/starhook/internal/git"

	"github.com/google/go-github/v28/github"
	"golang.org/x/sync/errgroup"
	"golang.org/x/sync/semaphore"
)

type result struct {
	updated bool
	created bool
}

type Service struct {
	gh     *gh.Client
	dir    string
	update bool
	store  internal.RepositoryStore
}

func NewService(ghClient *gh.Client, store internal.RepositoryStore, dir string) *Service {
	return &Service{
		gh:    ghClient,
		dir:   dir,
		store: store,
	}
}

// ListRepos lists all the repositories.
func (s *Service) ListRepos(ctx context.Context, query string) error {
	repos, err := s.store.FindRepos(ctx, internal.RepositoryFilter{}, internal.DefaultFindOptions)
	if err != nil {
		return err
	}

	lastUpdated := time.Time{}
	for _, repo := range repos {
		if repo.UpdatedAt.After(lastUpdated) {
			lastUpdated = repo.UpdatedAt
		}
	}

	fmt.Printf("==> local %d repositories (last synced: %s)\n", len(repos), humanize.Time(lastUpdated))
	return nil
}

// FetchRepos fetches and clones all the repositories.
func (s *Service) FetchRepos(ctx context.Context, query string) error {
	fmt.Println("==> fetching repositories")
	start := time.Now()

	repos, err := s.store.FindRepos(ctx, internal.RepositoryFilter{}, internal.DefaultFindOptions)
	if err != nil {
		return err
	}

	if len(repos) == 0 {
		return errors.New("no repositories to fetch, please sync first")
	}

	lastUpdated := time.Time{}
	for _, repo := range repos {
		if repo.UpdatedAt.After(lastUpdated) {
			lastUpdated = repo.UpdatedAt
		}
	}

	fmt.Printf("==> local %d repositories. last synced: %s\n", len(repos), humanize.Time(lastUpdated))

	if _, err := exec.LookPath("git"); err != nil {
		// make sure that `git` exists before we continue
		return errors.New("couldn't find 'git' in PATH")
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

	if len(clone) != 0 {
		fmt.Println("==> cloning repositories")
		if err := s.cloneRepos(ctx, clone); err != nil {
			return err
		}

		for _, repo := range clone {
			now := time.Now().UTC()
			err = s.store.UpdateRepo(ctx,
				internal.RepositoryBy{
					Name: &repo.Name,
				},
				internal.RepositoryUpdate{
					SyncedAt: &now,
				},
			)
		}
	}

	if len(update) != 0 {
		fmt.Println("==> updating repositories")
		if err := s.updateRepos(ctx, update); err != nil {
			return err
		}

		for _, repo := range update {
			now := time.Now().UTC()
			err = s.store.UpdateRepo(ctx,
				internal.RepositoryBy{
					Name: &repo.Name,
				},
				internal.RepositoryUpdate{
					SyncedAt: &now,
				},
			)
		}
	}

	if len(update) == 0 && len(clone) == 0 {
		fmt.Printf("==> everything is up-to-date (elapsed time: %s)\n",
			time.Since(start).String())
	} else {
		fmt.Printf("==> fetched and updated: %d repositories (elapsed time: %s)\n",
			len(repos), time.Since(start).String())
	}

	return nil
}

// SyncRepos syncs the remote repositories metadata with the store data.
func (s *Service) SyncRepos(ctx context.Context, query string) error {
	fmt.Println("==> syncing repositories")
	repos, err := s.store.FindRepos(ctx, internal.RepositoryFilter{}, internal.DefaultFindOptions)
	if err != nil {
		return err
	}

	localRepos := make(map[string]*internal.Repository, len(repos))
	for _, repo := range repos {
		localRepos[repo.Nwo] = repo
	}

	start := time.Now()
	ghRepos, err := s.gh.FetchRepos(ctx, query)
	if err != nil {
		return err
	}
	fetchedRepos := toRepos(ghRepos)

	fmt.Printf("==> queried %d repositories (elapsed time: %s)\n",
		len(fetchedRepos), time.Since(start).String())

	start = time.Now()
	g, ctx := errgroup.WithContext(ctx)

	const maxWorkers = 5
	sem := semaphore.NewWeighted(maxWorkers)

	done := make(chan struct{})
	ch := make(chan result)

	updated := 0
	created := 0
	go func() {
		for r := range ch {
			if r.updated {
				updated++
			}
			if r.created {
				created++
			}
		}
		close(done)
	}()

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
			updatedAt, err := s.gh.BranchTime(ctx, repo.Owner, repo.Name, repo.Branch)
			if err != nil {
				return err
			}
			repo.BranchUpdatedAt = updatedAt

			res := result{}
			localRepo, ok := localRepos[repo.Nwo]
			if !ok {
				fmt.Printf("  %q is created\n", repo.Name)
				_, err = s.store.CreateRepo(ctx, repo)
				if err != nil {
					return err
				}
				res.created = true
			} else if !localRepo.BranchUpdatedAt.Equal(repo.BranchUpdatedAt) {
				fmt.Printf("  %q is updated (last updated: %s)\n",
					repo.Name, humanize.Time(localRepo.BranchUpdatedAt))
				err = s.store.UpdateRepo(ctx,
					internal.RepositoryBy{
						Name: &repo.Name,
					},
					internal.RepositoryUpdate{
						BranchUpdatedAt: &repo.BranchUpdatedAt,
					},
				)
				if err != nil {
					return err
				}
				res.updated = true
			}

			select {
			case ch <- res:
			case <-ctx.Done():
				return ctx.Err()
			}

			return nil
		})
	}

	// Check whether any of the goroutines failed. Since g is accumulating the
	// errors, we don't need to send them (or check for them) in the individual
	// results sent on the channel.
	err = g.Wait()
	close(ch)
	if err != nil {
		return err
	}
	<-done

	if updated == 0 && created == 0 {
		fmt.Printf("==> everything is up-to-date (elapsed time: %s)\n", time.Since(start).String())
	}

	if updated != 0 {
		fmt.Printf("==> updated: %d repositories (elapsed time: %s)\n",
			updated, time.Since(start).String())
	}
	if created != 0 {
		fmt.Printf("==> created: %d repositories (elapsed time: %s)\n",
			created, time.Since(start).String())
	}

	return nil
}

func (s *Service) updateRepos(ctx context.Context, repos []*internal.Repository) error {
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
			return s.updateGitRepo(ctx, repo)
		})
	}

	if err := g.Wait(); err != nil {
		fmt.Printf("g.Wait() err = %+v\n", err)
	}

	fmt.Printf("==> updated to HEAD: %d repositories (elapsed time: %s)\n",
		len(repos), time.Since(start).String())
	return nil
}

func (s *Service) updateGitRepo(ctx context.Context, repo *internal.Repository) error {
	fmt.Printf("  updating %s\n", repo.Name)
	repoDir := filepath.Join(s.dir, repo.Name)
	g := &git.Client{Dir: repoDir}

	if _, err := g.Run("reset", "--hard"); err != nil {
		return err
	}
	if _, err := g.Run("clean", "-df"); err != nil {
		return err
	}

	branch, err := g.Run("rev-parse", "--abbrev-ref", "HEAD")
	if err != nil {
		return err
	}

	_, err = g.Run("pull", "origin", strings.TrimSpace(string(branch)))
	if err != nil {
		return err
	}

	return nil
}

func (s *Service) cloneRepos(ctx context.Context, repos []*internal.Repository) error {
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

func toRepos(rps []github.Repository) []*internal.Repository {
	repos := make([]*internal.Repository, 0, len(rps))
	for _, repo := range rps {
		owner := repo.GetOwner().GetLogin()
		name := repo.GetName()

		repos = append(repos, &internal.Repository{
			Nwo:    fmt.Sprintf("%s/%s", owner, name),
			Owner:  owner,
			Name:   name,
			Branch: repo.GetDefaultBranch(),
		})
	}

	return repos
}
