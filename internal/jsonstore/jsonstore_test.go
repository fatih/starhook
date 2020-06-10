package jsonstore

import (
	"context"
	"encoding/json"
	"io/ioutil"
	"os"
	"testing"

	"github.com/fatih/starhook/internal"
	"github.com/fatih/starhook/internal/testutil"
)

func TestNewRepositoryStore_newFile(t *testing.T) {
	dir, err := ioutil.TempDir("", "starhook")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		os.RemoveAll(dir)
	})

	store, err := NewRepositoryStore(dir)
	if err != nil {
		t.Fatal(err)
	}

	out, err := ioutil.ReadFile(store.path)
	if err != nil {
		t.Fatal(err)
	}

	var db internalDB
	err = json.Unmarshal(out, &db)
	if err != nil {
		t.Fatal(err)
	}

	testutil.Equals(t, len(db.Repositories), 0)
}

func TestNewRepositoryStore_existingFile(t *testing.T) {
	dir, err := ioutil.TempDir("", "starhook")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		os.RemoveAll(dir)
	})

	store, err := NewRepositoryStore(dir)
	if err != nil {
		t.Fatal(err)
	}

	content := `{"foo":"bar"}`
	err = ioutil.WriteFile(store.path, []byte(content), 0666)
	if err != nil {
		t.Fatal(err)
	}

	// open again and check whether file content is changed or not
	_, err = NewRepositoryStore(dir)
	if err != nil {
		t.Fatal(err)
	}

	out, err := ioutil.ReadFile(store.path)
	if err != nil {
		t.Fatal(err)
	}

	var db internalDB
	err = json.Unmarshal(out, &db)
	if err != nil {
		t.Fatal(err)
	}

	testutil.Equals(t, string(out), content)
}

func TestNewRepositoryStore_CreateRepo(t *testing.T) {
	dir, err := ioutil.TempDir("", "starhook")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		os.RemoveAll(dir)
	})

	store, err := NewRepositoryStore(dir)
	if err != nil {
		t.Fatal(err)
	}

	repo := &internal.Repository{
		Owner: "fatih",
		Name:  "vim-go",
	}

	ctx := context.Background()
	id, err := store.CreateRepo(ctx, repo)

	testutil.Ok(t, err)
	testutil.Equals(t, id, int64(1))

	out, err := ioutil.ReadFile(store.path)
	if err != nil {
		t.Fatal(err)
	}

	var db internalDB
	err = json.Unmarshal(out, &db)
	if err != nil {
		t.Fatal(err)
	}

	// the number of repos should be one
	testutil.Equals(t, len(db.Repositories), 1)
}

func TestNewRepositoryStore_CreateRepo_multiple(t *testing.T) {
	dir, err := ioutil.TempDir("", "starhook")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		os.RemoveAll(dir)
	})

	store, err := NewRepositoryStore(dir)
	if err != nil {
		t.Fatal(err)
	}

	repo := &internal.Repository{
		Owner: "fatih",
		Name:  "vim-go",
	}

	ctx := context.Background()

	id1, err := store.CreateRepo(ctx, repo)
	testutil.Ok(t, err)

	id2, err := store.CreateRepo(ctx, repo)
	testutil.Ok(t, err)

	id3, err := store.CreateRepo(ctx, repo)
	testutil.Ok(t, err)

	testutil.Equals(t, id1, int64(1))
	testutil.Equals(t, id2, int64(2))
	testutil.Equals(t, id3, int64(3))

	out, err := ioutil.ReadFile(store.path)
	if err != nil {
		t.Fatal(err)
	}

	var db internalDB
	err = json.Unmarshal(out, &db)
	if err != nil {
		t.Fatal(err)
	}

	testutil.Equals(t, len(db.Repositories), 3)
}

func TestNewRepositoryStore_FindRepos(t *testing.T) {
	dir, err := ioutil.TempDir("", "starhook")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		os.RemoveAll(dir)
	})

	store, err := NewRepositoryStore(dir)
	if err != nil {
		t.Fatal(err)
	}

	repo := &internal.Repository{
		Owner: "fatih",
		Name:  "vim-go",
	}

	ctx := context.Background()

	_, err = store.CreateRepo(ctx, repo)
	testutil.Ok(t, err)
	_, err = store.CreateRepo(ctx, repo)
	testutil.Ok(t, err)
	_, err = store.CreateRepo(ctx, repo)
	testutil.Ok(t, err)

	repos, err := store.FindRepos(ctx, internal.RepositoryFilter{}, internal.DefaultFindOptions)
	testutil.Ok(t, err)

	testutil.Equals(t, len(repos), 3)
}

func TestNewRepositoryStore_UpdateRepo(t *testing.T) {
	dir, err := ioutil.TempDir("", "starhook")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		os.RemoveAll(dir)
	})

	store, err := NewRepositoryStore(dir)
	if err != nil {
		t.Fatal(err)
	}

	repo := &internal.Repository{
		Owner: "fatih",
		Name:  "vim-go",
	}

	ctx := context.Background()
	id, err := store.CreateRepo(ctx, repo)
	testutil.Ok(t, err)

	err = store.UpdateRepo(ctx, internal.RepositoryBy{RepoID: &id}, internal.RepositoryUpdate{})
	testutil.Ok(t, err)

	rp, err := store.FindRepo(ctx, id)
	testutil.Ok(t, err)

	testutil.Equals(t, rp.Owner, repo.Owner)
	testutil.Equals(t, rp.Name, repo.Name)
	testutil.Assert(t, rp.UpdatedAt.After(rp.CreatedAt), "updated_at should be updated and should have a timestamp after created_at")

}

func TestNewRepositoryStore_FindRepo(t *testing.T) {
	dir, err := ioutil.TempDir("", "starhook")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		os.RemoveAll(dir)
	})

	store, err := NewRepositoryStore(dir)
	if err != nil {
		t.Fatal(err)
	}

	repo := &internal.Repository{
		Owner: "fatih",
		Name:  "vim-go",
	}

	ctx := context.Background()
	id, err := store.CreateRepo(ctx, repo)
	testutil.Ok(t, err)

	rp, err := store.FindRepo(ctx, id)
	testutil.Ok(t, err)
	testutil.Equals(t, rp.Owner, repo.Owner)
	testutil.Equals(t, rp.Name, repo.Name)
	testutil.Assert(t, !rp.CreatedAt.IsZero(), "created_at should be not zero")
	testutil.Assert(t, !rp.UpdatedAt.IsZero(), "updated_at should be not zero")
}
