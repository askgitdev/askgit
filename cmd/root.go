package cmd

import (
	"bufio"
	"database/sql"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/augmentable-dev/gitqlite/pkg/gitqlite"
	"github.com/gitsight/go-vcsurl"
	"github.com/go-git/go-git/v5"
	"github.com/olekukonko/tablewriter"
	"github.com/spf13/cobra"
)

//define flags in here
var (
	repo   string
	format string
)

func init() {
	rootCmd.PersistentFlags().StringVar(&repo, "repo", ".", "path to git repository (defaults to current directory). A remote git repository may also be specified, it will be cloned to a temporary directory on disk before query execution. Default git credentials are used for non-public remote repos.")
	rootCmd.PersistentFlags().StringVar(&format, "format", "table", "specify the output format. Options are 'csv' 'tsv' 'table' and 'json'")
}

func handleError(err error) {
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

var rootCmd = &cobra.Command{
	Use: `gitqlite "SELECT * FROM commits"`,
	Long: `
		gitqlite is a CLI for querying git repositories with SQL, using SQLite virtual tables.
		Example queries can be found in the GitHub repo: https://github.com/augmentable-dev/gitqlite`,
	Short: `query your github repos with SQL`,
	Run: func(cmd *cobra.Command, args []string) {
		info, err := os.Stdin.Stat()
		handleError(err)

		var query string
		if len(args) > 0 {
			query = args[0]
		} else if info.Mode()&os.ModeCharDevice == 0 {
			query, err = readStdin()
			handleError(err)
		} else {
			err = cmd.Help()
			handleError(err)
			os.Exit(0)
		}

		cwd, err := os.Getwd()
		handleError(err)

		// if a repo path is not supplied either as a flag or the second argument, use the current directory
		if repo == "" {
			if len(args) > 1 {
				repo = args[1]
			} else {
				repo = cwd
			}
		}

		// if the repo can be parsed as a remote git url, clone it to a temporary directory and use that as the repo path
		if remote, err := vcsurl.Parse(repo); err == nil { // if it can be parsed
			if r, err := remote.Remote(vcsurl.HTTPS); err == nil { // if it can be resolved into an HTTPS remote
				dir, err := ioutil.TempDir("", "repo")
				handleError(err)

				_, err = git.PlainClone(dir, false, &git.CloneOptions{
					URL: r,
				})
				handleError(err)

				defer func() {
					err := os.RemoveAll(dir)
					handleError(err)
				}()

				repo = dir
			}
		}

		repo, err = filepath.Abs(repo)
		if err != nil {
			handleError(err)
		}

		g, err := gitqlite.New(repo)
		handleError(err)

		rows, err := g.DB.Query(query)
		handleError(err)

		defer rows.Close()
		err = displayDB(rows)
		handleError(err)
	},
}

// Execute runs the root command
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		handleError(err)
	}
}

func displayDB(rows *sql.Rows) error {

	switch format {
	case "csv":
		err := csvDisplay(rows, ',')
		if err != nil {
			return err
		}
	case "tsv":
		err := csvDisplay(rows, '\t')
		if err != nil {
			return err
		}
	case "json":
		err := jsonDisplay(rows)
		if err != nil {
			return err
		}
	default:
		if shouldUseTable(rows) {
			err := tableDisplay(rows)
			if err != nil {
				return err
			}
		} else {
			err := csvDisplay(rows, ',')
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func csvDisplay(rows *sql.Rows, commaChar rune) error {

	columns, err := rows.Columns()
	if err != nil {
		return err
	}
	w := csv.NewWriter(os.Stdout)
	w.Comma = commaChar

	err = w.Write(columns)
	if err != nil {
		return err
	}
	pointers := make([]interface{}, len(columns))
	container := make([]sql.NullString, len(columns))

	for i := range pointers {
		pointers[i] = &container[i]
	}
	for rows.Next() {
		err := rows.Scan(pointers...)
		if err != nil {
			return err
		}

		r := make([]string, len(columns))
		for i, c := range container {
			if c.Valid {
				r[i] = c.String
			}
		}

		err = w.Write(r)
		if err != nil {
			return err
		}
	}
	w.Flush()
	return nil
}

func jsonDisplay(rows *sql.Rows) error {
	columns, err := rows.Columns()
	if err != nil {
		return err
	}

	values := make([]interface{}, len(columns))
	for i := range values {
		values[i] = new(interface{})
	}

	enc := json.NewEncoder(os.Stdout)

	for rows.Next() {
		err = rows.Scan(values...)
		if err != nil {
			return err
		}

		dest := make(map[string]interface{})

		for i, column := range columns {
			dest[column] = *(values[i].(*interface{}))
		}

		err := enc.Encode(dest)
		if err != nil {
			return err
		}

	}

	return nil
}
func tableDisplay(rows *sql.Rows) error {
	columns, err := rows.Columns()
	if err != nil {
		return err
	}
	pointers := make([]interface{}, len(columns))
	container := make([]sql.NullString, len(columns))

	for i := range pointers {
		pointers[i] = &container[i]
	}
	table := tablewriter.NewWriter(os.Stdout)
	table.SetHeader(columns)
	for rows.Next() {

		err := rows.Scan(pointers...)
		if err != nil {
			return err
		}

		r := make([]string, len(columns))
		for i, c := range container {
			if c.Valid {
				r[i] = c.String
			} else {
				r[i] = "NULL"
			}
		}

		table.Append(r)
		if err != nil {
			return err
		}
	}
	table.Render()
	return nil
}

func readStdin() (string, error) {
	reader := bufio.NewReader(os.Stdin)
	output, err := ioutil.ReadAll(reader)
	if err != nil {
		return "", err
	}
	returnString := string(output)
	return returnString, nil
}

// Switch between table and csv dependent on num columns
func shouldUseTable(rows *sql.Rows) bool {
	columns, err := rows.Columns()
	if err != nil {
		fmt.Println(err)
		return true //default to using a table
	}
	return len(columns) <= 5
}
