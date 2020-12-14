package gitqlite

import (
	"database/sql"
	"fmt"
	"io/ioutil"
	"os"
	"testing"

	git "github.com/libgit2/git2go/v31"
	"github.com/mattn/go-sqlite3"
)

var (
	fixtureRepoCloneURL = "https://github.com/augmentable-dev/tickgit"
	fixtureRepo         *git.Repository
	fixtureRepoDir      string
	fixtureDB           *sql.DB
)

func init() {
	sql.Register("gitqlite", &sqlite3.SQLiteDriver{
		ConnectHook: func(conn *sqlite3.SQLiteConn) error {
			err := conn.CreateModule("git_log", NewGitLogModule())
			if err != nil {
				return err
			}

			err = conn.CreateModule("git_log_cli", NewGitLogCLIModule())
			if err != nil {
				return err
			}

			err = conn.CreateModule("git_tree", NewGitFilesModule())
			if err != nil {
				return err
			}

			err = conn.CreateModule("git_tag", NewGitTagsModule())
			if err != nil {
				return err
			}

			err = conn.CreateModule("git_branch", NewGitBranchesModule())
			if err != nil {
				return err
			}
			err = conn.CreateModule("git_stats", NewGitStatsModule())
			if err != nil {
				return err
			}

			return nil
		},
	})
}

func TestMain(m *testing.M) {
	close, err := initFixtureRepo()
	if err != nil {
		panic(err)
	}
	err = initFixtureDB(fixtureRepoDir)
	if err != nil {
		panic(err)
	}
	code := m.Run()
	close()
	err = fixtureDB.Close()
	if err != nil {
		panic(err)
	}
	os.Exit(code)
}

func initFixtureRepo() (func() error, error) {
	dir, err := ioutil.TempDir("", "repo")
	if err != nil {
		return nil, err
	}

	fixtureRepo, err = git.Clone(fixtureRepoCloneURL, dir, &git.CloneOptions{})

	if err != nil {
		fmt.Println(err)
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

func initFixtureDB(repoPath string) error {
	db, err := sql.Open("gitqlite", ":memory:")
	if err != nil {
		return err
	}

	_, err = db.Exec(fmt.Sprintf("CREATE VIRTUAL TABLE IF NOT EXISTS commits USING git_log('%s');", repoPath))
	if err != nil {
		return err
	}

	_, err = db.Exec(fmt.Sprintf("CREATE VIRTUAL TABLE IF NOT EXISTS commits_cli USING git_log_cli('%s');", repoPath))
	if err != nil {
		return err
	}

	_, err = db.Exec(fmt.Sprintf("CREATE VIRTUAL TABLE IF NOT EXISTS stats USING git_stats('%s');", repoPath))
	if err != nil {
		return err
	}

	_, err = db.Exec(fmt.Sprintf("CREATE VIRTUAL TABLE IF NOT EXISTS files USING git_tree('%s');", repoPath))
	if err != nil {
		return err
	}
	_, err = db.Exec(fmt.Sprintf("CREATE VIRTUAL TABLE IF NOT EXISTS tags USING git_tag('%s');", repoPath))
	if err != nil {
		return err
	}
	_, err = db.Exec(fmt.Sprintf("CREATE VIRTUAL TABLE IF NOT EXISTS branches USING git_branch('%s');", repoPath))
	if err != nil {
		return err
	}

	fixtureDB = db
	return nil
}

func GetRowsCount(rows *sql.Rows) int {
	count := 0
	for rows.Next() {
		count++
	}

	return count
}

func GetRowContents(rows *sql.Rows) (colCount int, contents [][]string, err error) {
	columns, err := rows.Columns()
	if err != nil {
		return colCount, nil, err
	}

	pointers := make([]interface{}, len(columns))
	container := make([]sql.NullString, len(columns))

	for i := range pointers {
		pointers[i] = &container[i]
	}

	for rows.Next() {
		err = rows.Scan(pointers...)
		if err != nil {
			return colCount, nil, err
		}

		r := make([]string, len(columns))
		for i, c := range container {
			if c.Valid {
				r[i] = c.String
			} else {
				r[i] = "NULL"
			}
		}
		contents = append(contents, r)
	}
	return colCount, contents, err

}
