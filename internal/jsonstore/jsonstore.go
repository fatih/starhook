package jsonstore

import (
	"context"
	"encoding/json"
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
	reposfile := filepath.Join(dir, "repos.json")

	// check if the file exists
	_, err := ioutil.ReadFile(reposfile)
	if err != nil && !os.IsNotExist(err) {
		return nil, err
	}

	// if not, create a new one
	if os.IsNotExist(err) {
		if err := ioutil.WriteFile(reposfile, []byte(`{}`), 0666); err != nil {
			return nil, err
		}
	}

	return &RepositoryStore{
		path: reposfile,
	}, nil
}

func (r *RepositoryStore) FindRepos(ctx context.Context) ([]*internal.Repository, error) {
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
		db.CreatedAt = time.Now() //created the first time
	} else {
		repo.ID = int64(len(db.Repositories)) + 1
	}

	db.Repositories = append(db.Repositories, repo)
	db.UpdatedAt = time.Now().UTC()

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