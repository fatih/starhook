package jsonstore

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/fatih/starhook/internal"
)

const dbFile = "starhook.json"

var _ internal.RepositoryStore = (*RepositoryStore)(nil)

type internalDB struct {
	Repositories []*internal.Repository `json:"repositories"`
	CreatedAt    time.Time              `json:"created_at"`
	UpdatedAt    time.Time              `json:"updated_at"`
}

type RepositoryStore struct {
	path string
	mu   sync.Mutex
}

func NewRepositoryStore(dir string) (*RepositoryStore, error) {
	return newRepositoryStore(dir, time.Now().UTC)
}

func newRepositoryStore(dir string, nowFn func() time.Time) (*RepositoryStore, error) {
	if _, err := os.Stat(dir); err != nil {
		return nil, fmt.Errorf("dir %q does not exist", err)
	}

	reposfile := filepath.Join(dir, dbFile)

	// check if the file exists
	_, err := ioutil.ReadFile(reposfile)
	if err != nil && !os.IsNotExist(err) {
		return nil, err
	}

	// if not, create a new one
	if os.IsNotExist(err) {
		now := nowFn()
		db := internalDB{
			CreatedAt: now,
			UpdatedAt: now,
		}

		out, err := json.MarshalIndent(&db, " ", "  ")
		if err != nil {
			return nil, err
		}

		if err := ioutil.WriteFile(reposfile, []byte(out), 0666); err != nil {
			return nil, err
		}
	}

	return &RepositoryStore{
		path: reposfile,
	}, nil
}

func (r *RepositoryStore) FindRepos(ctx context.Context, filter internal.RepositoryFilter, opt internal.FindOptions) ([]*internal.Repository, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	in, err := ioutil.ReadFile(r.path)
	if err != nil {
		return nil, err
	}

	var db internalDB
	err = json.Unmarshal(in, &db)
	if err != nil {
		return nil, err
	}

	return db.Repositories, nil
}

func (r *RepositoryStore) FindRepo(ctx context.Context, repoID int64) (*internal.Repository, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	in, err := ioutil.ReadFile(r.path)
	if err != nil {
		return nil, err
	}

	var db internalDB
	err = json.Unmarshal(in, &db)
	if err != nil {
		return nil, err
	}

	for _, repo := range db.Repositories {
		if repo.ID == repoID {
			return repo, nil
		}
	}

	return nil, internal.ErrNotFound
}

func (r *RepositoryStore) CreateRepo(ctx context.Context, repo *internal.Repository) (int64, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	in, err := ioutil.ReadFile(r.path)
	if err != nil {
		return 0, err
	}

	var db internalDB
	err = json.Unmarshal(in, &db)
	if err != nil {
		return 0, err
	}

	if len(db.Repositories) == 0 {
		repo.ID = 1
	} else {
		repo.ID = int64(len(db.Repositories)) + 1
	}

	now := time.Now().UTC()
	repo.CreatedAt = now
	repo.UpdatedAt = now

	db.Repositories = append(db.Repositories, repo)
	db.UpdatedAt = now

	out, err := json.MarshalIndent(&db, " ", "  ")
	if err != nil {
		return 0, err
	}

	if err := ioutil.WriteFile(r.path, out, 0644); err != nil {
		return 0, err
	}

	return repo.ID, nil
}

func (r *RepositoryStore) UpdateRepo(ctx context.Context, repoID int64, upd internal.RepositoryUpdate) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	in, err := ioutil.ReadFile(r.path)
	if err != nil {
		return err
	}

	var db internalDB
	err = json.Unmarshal(in, &db)
	if err != nil {
		return err
	}

	for i, repo := range db.Repositories {
		if repo.ID == repoID {
			repo.UpdatedAt = time.Now()
			db.Repositories[i] = repo
		}
	}

	out, err := json.MarshalIndent(&db, " ", "  ")
	if err != nil {
		return err
	}

	if err := ioutil.WriteFile(r.path, out, 0644); err != nil {
		return err
	}

	return nil
}
