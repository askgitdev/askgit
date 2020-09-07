package gitqlite

import (
	"testing"

	"github.com/go-git/go-git/v5"
)

func TestGoGitStats(t *testing.T) {
	instance, err := New(fixtureRepoDir, &Options{SkipGitCLI: true})
	if err != nil {
		t.Fatal(err)
	}

	headRef, err := fixtureRepo.Head()
	if err != nil {
		t.Fatal(err)
	}
	commitChecker, err := fixtureRepo.Log(&git.LogOptions{
		From:  headRef.Hash(),
		Order: git.LogOrderCommitterTime,
	})
	if err != nil {
		t.Fatal(err)
	}
	commit, err := commitChecker.Next()
	if err != nil {
		t.Fatal(err)
	}
	stats, err := commit.Stats()
	if err != nil {
		t.Fatal(err)
	}
	vc := StatsCursor{repo: fixtureRepo, current: commit, commitIter: commitChecker, stats: stats, statIndex: 0}
	rows, err := instance.DB.Query("SELECT * FROM stats")
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	columns, err := rows.Columns()
	if err != nil {
		t.Fatal(err)
	}

	expected := 4
	if len(columns) != expected {
		t.Fatalf("expected %d columns, got: %d", expected, len(columns))
	}
	//for some reason rows close after above statement and ya gotta query again... and create the db again -_-
	instance, err = New(fixtureRepoDir, &Options{SkipGitCLI: true})
	if err != nil {
		t.Fatal(err)
	}

	//value checking for the commit_id and file name
	rows, err = instance.DB.Query("SELECT commit_id, file FROM stats")
	if err != nil {
		t.Fatal(err)
	}
	rowNum, contents, err := GetContents(rows)
	if err != nil {
		t.Fatalf("err %d at row Number %d", err, rowNum)
	}
	//for the range of contents check against each stat to see if there are discrepancies
	for i, c := range contents {
		if vc.current.ID().String() != c[0] {
			t.Fatalf("expected %s at row %d got %s", vc.current.ID().String(), i, c[0])
		}
		if vc.stats[vc.statIndex].Name != c[1] && c[1] != "NULL" && vc.stats[vc.statIndex].Name != "" {
			t.Fatalf("expected %s, at row %d got %s", vc.stats[vc.statIndex].Name, i, c[1])
		}

		err = vc.Next()
		if err != nil {
			t.Fatal(err)
		}

	}

}
func BenchmarkGoGitstatsCounts(b *testing.B) {
	for i := 0; i < b.N; i++ {
		instance, err := New(fixtureRepoDir, &Options{SkipGitCLI: true})
		if err != nil {
			b.Fatal(err)
		}
		rows, err := instance.DB.Query("SELECT * FROM stats")
		if err != nil {
			b.Fatal(err)
		}
		rowNum, _, err := GetContents(rows)
		if err != nil {
			b.Fatalf("err %d at row Number %d", err, rowNum)
		}
	}
}