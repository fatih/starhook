package starhook

import (
	"context"
	"errors"
	"fmt"
	"log"
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
	done := make(chan bool, 0)
	go func() {
		ghRepos, err := s.gh.FetchRepos(ctx, query)
		if err != nil {
			log.Printf("ERROR: fetching repositories has failed: %v\n", err)
		}

		fmt.Printf("==> remote %d repositories\n", len(ghRepos))
		close(done)
	}()

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

	fmt.Printf("==> local %d repositories (last updated: %s)\n", len(repos), humanize.Time(lastUpdated))

	<-done

	return nil
}

// FetchRepos fetches and clones all the repositories.
func (s *Service) FetchRepos(ctx context.Context, query string) error {
	fmt.Println("==> fetching repositories")
	start := time.Now()

	ghRepos, err := s.gh.FetchRepos(ctx, query)
	if err != nil {
		return err
	}

	fmt.Printf("==> found: %d repositories (elapsed time: %s)\n",
		len(ghRepos), time.Since(start).String())

	if err := s.cloneRepos(ctx, ghRepos); err != nil {
		return err
	}

	repos := toRepos(ghRepos)
	for _, repo := range repos {
		updatedAt, err := s.gh.BranchTime(ctx, repo.Owner, repo.Name, repo.Branch)
		if err != nil {
			return err
		}
		repo.BranchUpdatedAt = updatedAt

		_, err = s.store.CreateRepo(ctx, repo)
		if err != nil {
			return err
		}
	}

	return nil
}

func (s *Service) UpdateRepos(ctx context.Context, query string) error {
	fmt.Println("==> updating repositories")

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

	fmt.Printf("==> queried: %d repositories (elapsed time: %s)\n",
		len(fetchedRepos), time.Since(start).String())

	start = time.Now()
	totalUpdated := 0
	for i, repo := range fetchedRepos {
		localRepo, ok := localRepos[repo.Nwo]
		if !ok {
			// repo doesn't exist locally, clone it
			// TODO(arslan): git clone the repo
		} else {
			// repo exist. Check if it's outdated
			// NOTE(fatih): there is the possibility that the default branch
			// might have changed, for now we assume that's not the case, but
			// it's worth noting here.
			updatedAt, err := s.gh.BranchTime(ctx, repo.Owner, repo.Name, repo.Branch)
			if err != nil {
				return err
			}
			repo.BranchUpdatedAt = updatedAt
			fetchedRepos[i] = repo

			if localRepo.BranchUpdatedAt.Equal(updatedAt) {
				continue // nothing to do
			}

			totalUpdated++
			if localRepo.BranchUpdatedAt.Before(updatedAt) {
				fmt.Printf("  %q is updated (last updated: %s)", repo.Name, localRepo.BranchUpdatedAt)
			}

			// TODO(arslan): git checkout the repo
			err = s.store.UpdateRepo(ctx,
				internal.RepositoryBy{
					Name: &repo.Name,
				},
				internal.RepositoryUpdate{
					BranchUpdatedAt: &updatedAt,
				},
			)
			if err != nil {
				return err
			}
		}
	}

	if totalUpdated == 0 {
		fmt.Printf("==> everything is up-to-date (elapsed time: %s)\n", time.Since(start).String())
	} else {
		fmt.Printf("==> updated: %d repositories (elapsed time: %s)\n",
			totalUpdated, time.Since(start).String())
	}

	return nil
}

func (s *Service) updateRepo(ctx context.Context, repo github.Repository) error {
	fmt.Printf("  updating %s\n", repo.GetName())
	repoDir := filepath.Join(s.dir, repo.GetName())
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

func (s *Service) cloneRepos(ctx context.Context, repos []github.Repository) error {
	if _, err := exec.LookPath("git"); err != nil {
		// make sure that `git` exists before we continue
		return errors.New("couldn't find 'git' in PATH")
	}

	fmt.Println("==> cloning repositories")
	start := time.Now()

	// download at max 10 repos at the same time to not overload and burst the
	// server. Also makes it easier
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

func (s *Service) cloneRepo(ctx context.Context, repo github.Repository) error {
	repoDir := filepath.Join(s.dir, repo.GetName())

	// do not clone if it exists
	if _, err := os.Stat(repoDir); err == nil {
		return nil
	}

	fmt.Printf("  cloning %s\n", repo.GetName())
	g := &git.Client{}
	_, err := g.Run("clone", repo.GetCloneURL(), "--depth=1", repoDir)
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
			Nwo:             fmt.Sprintf("%s/%s", owner, name),
			Owner:           owner,
			Name:            name,
			Branch:          repo.GetDefaultBranch(),
			BranchUpdatedAt: time.Time{}, // TODO(fatih): fix this
		})
	}

	return repos
}

// func (c *Client) Run(ctx context.Context) error {
// 	fmt.Println("==> searching and fetching repositories")
// 	start := time.Now()

// 	var repos []github.Repository
// 	reposfile := filepath.Join(c.dir, "repos.json")
// 	out, err := ioutil.ReadFile(reposfile)
// 	if err != nil {
// 		if c.update {
// 			return fmt.Errorf("no repos.json file found in dir %q. Please remove the --update flag", c.dir)
// 		}

// 		repos, err = c.gh.FetchRepos(ctx, c.query)
// 		if err != nil {
// 			return err
// 		}

// 		// dump data so we don't fetch it again
// 		out, err := json.MarshalIndent(repos, " ", " ")
// 		if err != nil {
// 			return err
// 		}

// 		if err := ioutil.WriteFile(reposfile, out, 0644); err != nil {
// 			return err
// 		}
// 	} else {
// 		// load from cached file
// 		if err := json.Unmarshal(out, &repos); err != nil {
// 			return err
// 		}

// 		fmt.Printf("==> repos.json found: %d repositories (elapsed time: %s)\n",
// 			len(repos), time.Since(start).String())

// 	}

// 	if c.update {
// 		fmt.Println("==> updating repositories ...")
// 		if err := c.updateRepos(ctx, reposfile, c.query, repos); err != nil {
// 			return err
// 		}
// 	} else {
// 		if err := c.cloneRepos(ctx, repos); err != nil {
// 			return err
// 		}
// 	}

// 	return nil
// }
