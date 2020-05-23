package jsonstore

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"testing"
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

	equals(t, string(out), `{}`)
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

// equals fails the test if exp is not equal to act.
func equals(tb testing.TB, exp, act interface{}) {
	if !reflect.DeepEqual(exp, act) {
		_, file, line, _ := runtime.Caller(1)
		fmt.Printf("\033[31m%s:%d:\n\n\texp: %#v\n\n\tgot: %#v\033[39m\n\n", filepath.Base(file), line, exp, act)
		tb.FailNow()
	}
}
