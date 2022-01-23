package jsonstore

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/fatih/starhook/internal"
)

const dbFile = "starhook.json"

var _ internal.MetadataStore = (*MetadataStore)(nil)

type internalDB struct {
	Repositories []*internal.Repository `json:"repositories"`

	// Query defines the initial GitHub search query to establish the DB
	Query string `json:"query"`
}

type MetadataStore struct {
	path string
	mu   sync.Mutex
}

func NewMetadataStore(dir, query string) (*MetadataStore, error) {
	if _, err := os.Stat(dir); err != nil {
		fmt.Printf("err = %+v\n", err)
		return nil, fmt.Errorf("dir %q does not exist", dir)
	}

	reposfile := filepath.Join(dir, dbFile)

	// check if the file exists
	in, err := os.ReadFile(reposfile)
	if err != nil && !os.IsNotExist(err) {
		return nil, err
	}

	// if not, create a new one
	if os.IsNotExist(err) {
		db := internalDB{
			Query: query,
		}

		out, err := json.MarshalIndent(&db, " ", "  ")
		if err != nil {
			return nil, err
		}

		if err := os.WriteFile(reposfile, []byte(out), 0666); err != nil {
			return nil, err
		}
	} else {
		// check whether the query matches
		var db internalDB
		err = json.Unmarshal(in, &db)
		if err != nil {
			return nil, err
		}

		if db.Query != query {
			return nil, fmt.Errorf("store error: query mismatch\n  current: %q\n  passed : %q",
				db.Query, query)
		}
	}

	return &MetadataStore{
		path: reposfile,
	}, nil
}

func (r *MetadataStore) FindRepos(ctx context.Context, filter internal.RepositoryFilter, opt internal.FindOptions) ([]*internal.Repository, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	in, err := os.ReadFile(r.path)
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

func (r *MetadataStore) FindRepo(ctx context.Context, repoID int64) (*internal.Repository, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	in, err := os.ReadFile(r.path)
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

func (r *MetadataStore) CreateRepo(ctx context.Context, repo *internal.Repository) (int64, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	in, err := os.ReadFile(r.path)
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

	out, err := json.MarshalIndent(&db, " ", "  ")
	if err != nil {
		return 0, err
	}

	if err := os.WriteFile(r.path, out, 0644); err != nil {
		return 0, err
	}

	return repo.ID, nil
}

func (r *MetadataStore) UpdateRepo(ctx context.Context, by internal.RepositoryBy, upd internal.RepositoryUpdate) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	in, err := os.ReadFile(r.path)
	if err != nil {
		return err
	}

	var db internalDB
	err = json.Unmarshal(in, &db)
	if err != nil {
		return err
	}

	updatable := func(repo *internal.Repository) bool {
		if by.Name != nil && *by.Name == repo.Name {
			return true
		}
		if by.RepoID != nil && *by.RepoID == repo.ID {
			return true
		}

		return false
	}

	for i, repo := range db.Repositories {
		if !updatable(repo) {
			continue
		}

		if upd.Nwo != nil {
			repo.Nwo = *upd.Nwo
		}

		if upd.Owner != nil {
			repo.Owner = *upd.Owner
		}

		if upd.SHA != nil {
			repo.SHA = *upd.SHA
		}

		if upd.BranchUpdatedAt != nil {
			repo.BranchUpdatedAt = *upd.BranchUpdatedAt
		}

		if upd.SyncedAt != nil {
			repo.SyncedAt = *upd.SyncedAt
		}

		repo.UpdatedAt = time.Now().UTC()
		db.Repositories[i] = repo
	}

	out, err := json.MarshalIndent(&db, " ", "  ")
	if err != nil {
		return err
	}

	if err := os.WriteFile(r.path, out, 0644); err != nil {
		return err
	}

	return nil
}

func (r *MetadataStore) DeleteRepo(ctx context.Context, by internal.RepositoryBy) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	in, err := os.ReadFile(r.path)
	if err != nil {
		return err
	}

	var db internalDB
	err = json.Unmarshal(in, &db)
	if err != nil {
		return err
	}

	deleteAble := func(repo *internal.Repository) bool {
		if by.Name != nil && *by.Name == repo.Name {
			return true
		}
		if by.RepoID != nil && *by.RepoID == repo.ID {
			return true
		}

		return false
	}

	ix := 0 // to be deleted
	for i, repo := range db.Repositories {
		if !deleteAble(repo) {
			continue
		}
		ix = i
	}

	db.Repositories = append(db.Repositories[:ix], db.Repositories[ix+1:]...)
	out, err := json.MarshalIndent(&db, " ", "  ")
	if err != nil {
		return err
	}

	if err := os.WriteFile(r.path, out, 0644); err != nil {
		return err
	}

	return nil
}
