package gitlog

import (
	"io"
	"io/ioutil"
	"os"
	"testing"

	git "github.com/libgit2/git2go/v30"
)

var (
	fixtureRepoCloneURL = "https://github.com/augmentable-dev/gitqlite"
	fixtureRepo         *git.Repository
	fixtureRepoDir      string
)

func TestMain(m *testing.M) {
	close, err := initFixtureRepo()
	if err != nil {
		panic(err)
	}
	code := m.Run()
	close()
	os.Exit(code)
}

func initFixtureRepo() (func() error, error) {
	dir, err := ioutil.TempDir("", "repo")
	if err != nil {
		return nil, err
	}

	fixtureRepo, err = git.Clone(fixtureRepoCloneURL, dir, &git.CloneOptions{})
	if err != nil {
		return nil, err
	}

	fixtureRepoDir = dir

	return func() error {
		err := os.RemoveAll(dir)
		if err != nil {
			return err
		}
		return nil
	}, nil
}

func TestParse(t *testing.T) {
	iter, err := Execute(fixtureRepoDir)
	if err != nil {
		t.Fatal(err)
	}

	count := 0
	for {
		commit, err := iter.Next()
		if err != nil {
			if err == io.EOF {
				break
			}
			t.Fatal(err)
		}
		if commit.SHA == "" {
			t.Fatal("expected SHA, got <empty string>")
		}
		count++
	}

	if err != nil {
		t.Fatal(err)
	}

	revWalk, err := fixtureRepo.Walk()
	if err != nil {
		t.Fatal(err)
	}
	defer revWalk.Free()

	err = revWalk.PushHead()
	if err != nil {
		t.Fatal(err)
	}

	shouldBeCount := 0
	err = revWalk.Iterate(func(*git.Commit) bool {
		shouldBeCount++
		return true
	})
	if err != nil {
		t.Fatal(err)
	}

	if count != shouldBeCount {
		t.Fatalf("incorrect number of commits, expected: %d got: %d", shouldBeCount, count)
	}
}
