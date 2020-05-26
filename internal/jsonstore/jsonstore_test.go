package jsonstore

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"testing"

	"github.com/fatih/starhook/internal"
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

	equals(t, len(db.Repositories), 0)
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

	equals(t, string(out), content)
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

	ok(t, err)
	equals(t, id, int64(1))

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
	equals(t, len(db.Repositories), 1)
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
	ok(t, err)

	id2, err := store.CreateRepo(ctx, repo)
	ok(t, err)

	id3, err := store.CreateRepo(ctx, repo)
	ok(t, err)

	equals(t, id1, int64(1))
	equals(t, id2, int64(2))
	equals(t, id3, int64(3))

	out, err := ioutil.ReadFile(store.path)
	if err != nil {
		t.Fatal(err)
	}

	var db internalDB
	err = json.Unmarshal(out, &db)
	if err != nil {
		t.Fatal(err)
	}

	equals(t, len(db.Repositories), 3)
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
	ok(t, err)
	_, err = store.CreateRepo(ctx, repo)
	ok(t, err)
	_, err = store.CreateRepo(ctx, repo)
	ok(t, err)

	repos, err := store.FindRepos(ctx, internal.RepositoryFilter{}, internal.DefaultFindOptions)
	ok(t, err)

	equals(t, len(repos), 3)
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
	ok(t, err)

	err = store.UpdateRepo(ctx, internal.RepositoryBy{RepoID: &id}, internal.RepositoryUpdate{})
	ok(t, err)

	rp, err := store.FindRepo(ctx, id)
	ok(t, err)

	equals(t, rp.Owner, repo.Owner)
	equals(t, rp.Name, repo.Name)
	assert(t, rp.UpdatedAt.After(rp.CreatedAt), "updated_at should be updated and should have a timestamp after created_at")

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
	ok(t, err)

	rp, err := store.FindRepo(ctx, id)
	ok(t, err)
	equals(t, rp.Owner, repo.Owner)
	equals(t, rp.Name, repo.Name)
	assert(t, !rp.CreatedAt.IsZero(), "created_at should be not zero")
	assert(t, !rp.UpdatedAt.IsZero(), "updated_at should be not zero")
}

// assert fails the test if the condition is false.
func assert(tb testing.TB, condition bool, msg string, v ...interface{}) {
	if !condition {
		_, file, line, _ := runtime.Caller(1)
		fmt.Printf("\033[31m%s:%d: "+msg+"\033[39m\n\n", append([]interface{}{filepath.Base(file), line}, v...)...)
		tb.FailNow()
	}
}

// ok fails the test if an err is not nil.
func ok(tb testing.TB, err error) {
	if err != nil {
		_, file, line, _ := runtime.Caller(1)
		fmt.Printf("\033[31m%s:%d: unexpected error: %s\033[39m\n\n", filepath.Base(file), line, err.Error())
		tb.FailNow()
	}
}

// equals fails the test if got is not equal to want.
func equals(tb testing.TB, got, want interface{}) {
	if !reflect.DeepEqual(got, want) {
		_, file, line, _ := runtime.Caller(1)
		fmt.Printf("\033[31m%s:%d:\n\n\texp: %#v\n\n\tgot: %#v\033[39m\n\n", filepath.Base(file), line, got, want)
		tb.FailNow()
	}
}
