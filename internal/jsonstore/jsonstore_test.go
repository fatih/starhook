package jsonstore

import (
	"context"
	"encoding/json"
	"io/ioutil"
	"testing"

	"github.com/fatih/starhook/internal"

	qt "github.com/frankban/quicktest"
)

func TestNewRepositoryStore_newFile(t *testing.T) {
	c := qt.New(t)
	dir := c.Mkdir()

	store, err := NewRepositoryStore(dir)
	c.Assert(err, qt.IsNil)

	out, err := ioutil.ReadFile(store.path)
	c.Assert(err, qt.IsNil)

	var db internalDB
	err = json.Unmarshal(out, &db)
	c.Assert(err, qt.IsNil)
	c.Assert(db.Repositories, qt.HasLen, 0)
}

func TestNewRepositoryStore_existingFile(t *testing.T) {
	c := qt.New(t)
	dir := c.Mkdir()

	store, err := NewRepositoryStore(dir)
	if err != nil {
		t.Fatal(err)
	}

	content := `{"foo":"bar"}`
	err = ioutil.WriteFile(store.path, []byte(content), 0666)
	c.Assert(err, qt.IsNil)

	// open again and check whether file content is changed or not
	_, err = NewRepositoryStore(dir)
	c.Assert(err, qt.IsNil)

	out, err := ioutil.ReadFile(store.path)
	c.Assert(err, qt.IsNil)

	var db internalDB
	err = json.Unmarshal(out, &db)
	c.Assert(err, qt.IsNil)
	c.Assert(string(out), qt.Equals, content)
}

func TestNewRepositoryStore_CreateRepo(t *testing.T) {
	c := qt.New(t)
	dir := c.Mkdir()

	store, err := NewRepositoryStore(dir)
	c.Assert(err, qt.IsNil)

	repo := &internal.Repository{
		Owner: "fatih",
		Name:  "vim-go",
	}

	ctx := context.Background()
	id, err := store.CreateRepo(ctx, repo)
	c.Assert(err, qt.IsNil)
	c.Assert(id, qt.Equals, int64(1))

	out, err := ioutil.ReadFile(store.path)
	c.Assert(err, qt.IsNil)

	var db internalDB
	err = json.Unmarshal(out, &db)
	c.Assert(err, qt.IsNil)

	// the number of repos should be one
	c.Assert(db.Repositories, qt.HasLen, 1)
}

func TestNewRepositoryStore_CreateRepo_multiple(t *testing.T) {
	c := qt.New(t)
	dir := c.Mkdir()

	store, err := NewRepositoryStore(dir)
	c.Assert(err, qt.IsNil)

	repo := &internal.Repository{
		Owner: "fatih",
		Name:  "vim-go",
	}

	ctx := context.Background()

	id1, err := store.CreateRepo(ctx, repo)
	c.Assert(err, qt.IsNil)

	id2, err := store.CreateRepo(ctx, repo)
	c.Assert(err, qt.IsNil)

	id3, err := store.CreateRepo(ctx, repo)
	c.Assert(err, qt.IsNil)
	c.Assert(id1, qt.DeepEquals, int64(1))
	c.Assert(id2, qt.DeepEquals, int64(2))
	c.Assert(id3, qt.DeepEquals, int64(3))

	out, err := ioutil.ReadFile(store.path)
	c.Assert(err, qt.IsNil)

	var db internalDB
	err = json.Unmarshal(out, &db)
	c.Assert(err, qt.IsNil)
	c.Assert(db.Repositories, qt.HasLen, 3)
}

func TestNewRepositoryStore_FindRepos(t *testing.T) {
	c := qt.New(t)
	dir := c.Mkdir()

	store, err := NewRepositoryStore(dir)
	c.Assert(err, qt.IsNil)

	repo := &internal.Repository{
		Owner: "fatih",
		Name:  "vim-go",
	}

	ctx := context.Background()

	_, err = store.CreateRepo(ctx, repo)
	c.Assert(err, qt.IsNil)
	_, err = store.CreateRepo(ctx, repo)
	c.Assert(err, qt.IsNil)
	_, err = store.CreateRepo(ctx, repo)
	c.Assert(err, qt.IsNil)

	repos, err := store.FindRepos(ctx, internal.RepositoryFilter{}, internal.DefaultFindOptions)
	c.Assert(err, qt.IsNil)
	c.Assert(repos, qt.HasLen, 3)
}

func TestNewRepositoryStore_UpdateRepo(t *testing.T) {
	c := qt.New(t)
	dir := c.Mkdir()

	store, err := NewRepositoryStore(dir)
	c.Assert(err, qt.IsNil)

	repo := &internal.Repository{
		Owner: "fatih",
		Name:  "vim-go",
	}

	ctx := context.Background()
	id, err := store.CreateRepo(ctx, repo)
	c.Assert(err, qt.IsNil)

	err = store.UpdateRepo(ctx, internal.RepositoryBy{RepoID: &id}, internal.RepositoryUpdate{})
	c.Assert(err, qt.IsNil)

	rp, err := store.FindRepo(ctx, id)
	c.Assert(err, qt.IsNil)

	c.Assert(rp.Owner, qt.DeepEquals, repo.Owner)
	c.Assert(rp.Name, qt.DeepEquals, repo.Name)
	c.Assert(rp.UpdatedAt.After(rp.CreatedAt), qt.IsTrue, qt.Commentf("updated_at should be updated and should have a timestamp after created_at"))

}

func TestNewRepositoryStore_FindRepo(t *testing.T) {
	c := qt.New(t)
	dir := c.Mkdir()

	store, err := NewRepositoryStore(dir)
	c.Assert(err, qt.IsNil)

	repo := &internal.Repository{
		Owner: "fatih",
		Name:  "vim-go",
	}

	ctx := context.Background()
	id, err := store.CreateRepo(ctx, repo)
	c.Assert(err, qt.IsNil)

	rp, err := store.FindRepo(ctx, id)
	c.Assert(err, qt.IsNil)
	c.Assert(rp.Owner, qt.Equals, repo.Owner)
	c.Assert(rp.Name, qt.Equals, repo.Name)
	c.Assert(!rp.CreatedAt.IsZero(), qt.IsTrue, qt.Commentf("created_at should be not zero"))
	c.Assert(!rp.UpdatedAt.IsZero(), qt.IsTrue, qt.Commentf("updated_at should be not zero"))
}

func TestNewRepositoryStore_DeleteRepo(t *testing.T) {
	c := qt.New(t)
	dir := c.Mkdir()

	store, err := NewRepositoryStore(dir)
	c.Assert(err, qt.IsNil)

	ctx := context.Background()

	repo := &internal.Repository{
		Owner: "fatih",
		Name:  "vim-go",
	}
	id, err := store.CreateRepo(ctx, repo)
	c.Assert(err, qt.IsNil)

	repo2 := &internal.Repository{
		Owner: "fatih",
		Name:  "gomodifytags",
	}
	_, err = store.CreateRepo(ctx, repo2)
	c.Assert(err, qt.IsNil)

	// delete first repo
	err = store.DeleteRepo(ctx, internal.RepositoryBy{RepoID: &id})
	c.Assert(err, qt.IsNil)

	repos, err := store.FindRepos(ctx, internal.RepositoryFilter{}, internal.DefaultFindOptions)
	c.Assert(err, qt.IsNil)
	c.Assert(repos, qt.HasLen, 1)
}
