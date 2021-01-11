[![GoDev](https://img.shields.io/static/v1?label=godev&message=reference&color=00add8)](https://pkg.go.dev/github.com/augmentable-dev/askgit)
[![BuildStatus](https://github.com/augmentable-dev/askgit/workflows/tests/badge.svg)](https://github.com/augmentable-dev/askgit/actions?workflow=tests)
[![Go Report Card](https://goreportcard.com/badge/github.com/augmentable-dev/askgit)](https://goreportcard.com/report/github.com/augmentable-dev/askgit)
[![TODOs](https://badgen.net/https/api.tickgit.com/badgen/github.com/augmentable-dev/askgit)](https://www.tickgit.com/browse?repo=github.com/augmentable-dev/askgit)
[![codecov](https://codecov.io/gh/augmentable-dev/askgit/branch/master/graph/badge.svg)](https://codecov.io/gh/augmentable-dev/askgit)


# askgit <!-- omit in toc -->

`askgit` is a command-line tool for running SQL queries on git repositories.
It's meant for ad-hoc querying of git repositories on disk through a common interface (SQL), as an alternative to patching together various shell commands.
It can execute queries that look like:
```sql
-- how many commits have been authored by user@email.com?
SELECT count(*) FROM commits WHERE author_email = 'user@email.com'
```

There's also preliminary support for executing queries against the GitHub API.

More in-depth examples and documentation can be found below.

- [Installation](#installation)
  - [Homebrew](#homebrew)
  - [Go](#go)
  - [Using Docker](#using-docker)
    - [Running commands](#running-commands)
    - [Running commands from STDIN](#running-commands-from-stdin)
- [Usage](#usage)
  - [Tables](#tables)
    - [Local Git Repository](#local-git-repository)
      - [`commits`](#commits)
      - [`stats`](#stats)
      - [`files`](#files)
      - [`branches`](#branches)
      - [`tags`](#tags)
    - [GitHub Tables](#github-tables)
      - [`github_org_repos` and `github_user_repos`](#github_org_repos-and-github_user_repos)
      - [`github_pull_requests`](#github_pull_requests)
      - [`github_issues`](#github_issues)
  - [Example Queries](#example-queries)
    - [Interactive mode](#interactive-mode)
    - [Exporting](#exporting)

## Installation

### Homebrew

```
brew tap augmentable-dev/askgit
brew install askgit
```

### Go

```
go get -v -tags=sqlite_vtable github.com/augmentable-dev/askgit
```

Will use the go tool chain to install a binary to `$GOBIN`.

```
GOBIN=$(pwd) go get -v -tags=sqlite_vtable github.com/augmentable-dev/askgit
```

Will produce a binary in your current directory.


### Using Docker

Build an image locally using docker

```
docker build -t askgit:latest .
```

Or use an official image from [docker hub](https://hub.docker.com/repository/docker/augmentable/askgit)

```
docker pull augmentable/askgit:latest
```

#### Running commands

`askgit` operates on a git repository. This repository needs to be attached as a volume. This example uses the (bash) built-in command `pwd` for the current working directory

> [**pwd**] Print the absolute pathname of the current working directory.

```
docker run --rm -v `pwd`:/repo:ro augmentable/askgit "SELECT * FROM commits"
```

#### Running commands from STDIN

For piping commands via STDIN, the docker command needs to be told to run non-interactively, as well as attaching the repository at `/repo`.

```
cat query.sql | docker run --rm -i -v `pwd`:/repo:ro augmentable/askgit
```

## Usage

```
askgit -h
```

Will output the most up to date usage instructions for your version of the CLI.
Typically the first argument is a SQL query string:

```
askgit "SELECT * FROM commits"
```

Your current working directory will be used as the path to the git repository to query by default.
Use the `--repo` flag to specify an alternate path, or even a remote repository reference (http(s) or ssh).
`askgit` will clone the remote repository to a temporary directory before executing a query.

You can also pass a query in via `stdin`:

```
cat query.sql | askgit
```

By default, output will be an ASCII table.
Use `--format json` or `--format csv` for alternatives.
See `-h` for all the options.

### Tables

#### Local Git Repository

When a repo is specified (either by the `--repo` flag or from the current directory), the following tables are available to query.

##### `commits`

Similar to `git log`, the `commits` table includes all commits in the history of the currently checked out commit.

| Column          | Type     |
|-----------------|----------|
| id              | TEXT     |
| message         | TEXT     |
| summary         | TEXT     |
| author_name     | TEXT     |
| author_email    | TEXT     |
| author_when     | DATETIME |
| committer_name  | TEXT     |
| committer_email | TEXT     |
| committer_when  | DATETIME |
| parent_id       | TEXT     |
| parent_count    | INT      |

##### `stats`

| Column    | Type |
|-----------|------|
| commit_id | TEXT |
| file      | TEXT |
| additions | INT  |
| deletions | INT  |

##### `files`

The `files` table iterates over _ALL_ the files in a commit history, by default from what's checked out in the repository.
The full table is every file in every tree of a commit history.
Use the `commit_id` column to filter for files that belong to the work tree of a specific commit.

| Column     | Type |
|------------|------|
| commit_id  | TEXT |
| name       | TEXT |
| contents   | TEXT |
| executable | BOOL |


##### `branches`

| Column | Type |
|--------|------|
| name   | TEXT |
| remote | BOOL |
| target | TEXT |
| head   | BOOL |

##### `tags`

| Column       | Type |
|--------------|------|
| full_name    | TEXT |
| name         | TEXT |
| lightweight  | BOOL |
| target       | TEXT |
| tagger_name  | TEXT |
| tagger_email | TEXT |
| message      | TEXT |
| target_type  | TEXT |

#### GitHub Tables

**This functionality is under development and likely to change**

The following tables make GitHub API requests to retrieve data during query execution.
As such, you should ensure the `GITHUB_TOKEN` environment variable is set so that API requests are authenticated.
Unauthenticated API requests (no `GITHUB_TOKEN`) are subject to a stricter rate limit by GitHub, and may take longer to execute (query execution will try to respect the applicable rate limit).

##### `github_org_repos` and `github_user_repos`

These tables can be queried as table-valued functions expecting a single parameter, like so:

```sql
-- return all repos from a github *org*
SELECT * FROM github_org_repos('augmentable-dev')

-- return all repos from a github *user*
SELECT * FROM github_user_repos('augmentable-dev')
```

| Column            | Type     |
|-------------------|----------|
| id                | INT      |
| node_id           | TEXT     |
| name              | TEXT     |
| full_name         | TEXT     |
| owner             | TEXT     |
| private           | BOOL     |
| description       | TEXT     |
| fork              | BOOL     |
| homepage          | TEXT     |
| language          | TEXT     |
| forks_count       | INT      |
| stargazers_count  | INT      |
| watchers_count    | INT      |
| size              | INT      |
| default_branch    | TEXT     |
| open_issues_count | INT      |
| topics            | TEXT     |
| has_issues        | BOOL     |
| has_projects      | BOOL     |
| has_wiki          | BOOL     |
| has_pages         | BOOL     |
| has_downloads     | BOOL     |
| archived          | BOOL     |
| pushed_at         | DATETIME |
| created_at        | DATETIME |
| updated_at        | DATETIME |
| permissions       | TEXT     |

##### `github_pull_requests`

This table expects 2 parameters, `github_pull_requests('augmentable-dev', 'askgit')`:

```sql
SELECT count(*) FROM github_pull_requests('augmentable-dev', 'askgit') WHERE state = 'open'
```

| Column                    | Type     |
|---------------------------|----------|
| id                        | INT      |
| node_id                   | TEXT     |
| number                    | INT      |
| state                     | TEXT     |
| locked                    | BOOL     |
| title                     | TEXT     |
| user_login                | TEXT     |
| body                      | TEXT     |
| labels                    | TEXT     |
| active_lock_reason        | TEXT     |
| created_at                | DATETIME |
| updated_at                | DATETIME |
| closed_at                 | DATETIME |
| merged_at                 | DATETIME |
| merge_commit_sha          | TEXT     |
| assignee_login            | TEXT     |
| assignees                 | TEXT     |
| requested_reviewer_logins | TEXT     |
| head_label                | TEXT     |
| head_ref                  | TEXT     |
| head_sha                  | TEXT     |
| head_repo_owner           | TEXT     |
| head_repo_name            | TEXT     |
| base_label                | TEXT     |
| base_ref                  | TEXT     |
| base_sha                  | TEXT     |
| base_repo_owner           | TEXT     |
| base_repo_name            | TEXT     |
| author_association        | TEXT     |
| merged                    | BOOL     |
| mergeable                 | BOOL     |
| mergeable_state           | BOOL     |
| merged_by_login           | TEXT     |
| comments                  | INT      |
| maintainer_can_modify     | BOOL     |
| commits                   | INT      |
| additions                 | INT      |
| deletions                 | INT      |
| changed_files             | INT      |

##### `github_issues`

This table expects 2 parameters, `github_issues('augmentable-dev', 'askgit')`:

```sql
SELECT count(*) FROM github_issues('augmentable-dev', 'askgit') WHERE state = 'open'
```


| Column                    | Type     |
|---------------------------|----------|
| id                        | INT      |
| node_id                   | TEXT     |
| number                    | INT      |
| state                     | TEXT     |
| locked                    | BOOL     |
| title                     | TEXT     |
| user_login                | TEXT     |
| body                      | TEXT     |
| labels                    | TEXT     |
| active_lock_reason        | TEXT     |
| created_at                | DATETIME |
| updated_at                | DATETIME |
| closed_at                 | DATETIME |
| merged_at                 | DATETIME |
| merge_commit_sha          | TEXT     |
| assignee_login            | TEXT     |
| assignees                 | TEXT     |
| url                       | TEXT     |
| html_url                  | TEXT     |
| comments_url              | TEXT     |
| events_url                | TEXT     |
| repository_url            | TEXT     |
| comments                  | INT      |
| milestone                 | TEXT     |
| reactions                 | INT      |


### Example Queries

This will return all commits in the history of the currently checked out branch/commit of the repo.
```sql
SELECT * FROM commits
```

Return the (de-duplicated) email addresses of commit authors:
```sql
SELECT DISTINCT author_email FROM commits
```

Return the commit counts of every author (by email):
```sql
SELECT author_email, count(*) FROM commits GROUP BY author_email ORDER BY count(*) DESC
```

Same as above, but excluding merge commits:
```sql
SELECT author_email, count(*) FROM commits WHERE parent_count < 2 GROUP BY author_email ORDER BY count(*) DESC
```

This is an expensive query.
It will iterate over every file in every tree of every commit in the current history:
```sql
SELECT * FROM files
```


Outputs the set of files in the tree of a certain commit:
```sql
SELECT * FROM files WHERE commit_id='some_commit_id'
```


Same as above if you just have the commit short id:
```sql
SELECT * FROM files WHERE commit_id LIKE 'shortened_commit_id%'
```


Returns author emails with lines added/removed, ordered by total number of commits in the history (excluding merges):
```sql
SELECT count(DISTINCT commits.id) AS commits, SUM(additions) AS additions, SUM(deletions) AS deletions, author_email
FROM commits LEFT JOIN stats ON commits.id = stats.commit_id
WHERE commits.parent_count < 2
GROUP BY author_email ORDER BY commits
```



Returns commit counts by author, broken out by day of the week:

```sql
SELECT
    count(*) AS commits,
    count(CASE WHEN strftime('%w',author_when)='0' THEN 1 END) AS sunday,
    count(CASE WHEN strftime('%w',author_when)='1' THEN 1 END) AS monday,
    count(CASE WHEN strftime('%w',author_when)='2' THEN 1 END) AS tuesday,
    count(CASE WHEN strftime('%w',author_when)='3' THEN 1 END) AS wednesday,
    count(CASE WHEN strftime('%w',author_when)='4' THEN 1 END) AS thursday,
    count(CASE WHEN strftime('%w',author_when)='5' THEN 1 END) AS friday,
    count(CASE WHEN strftime('%w',author_when)='6' THEN 1 END) AS saturday,
    author_email
FROM commits GROUP BY author_email ORDER BY commits
```


#### Interactive mode
```
askgit --interactive
```

Will display a basic terminal UI for composing and executing queries, powered by [gocui](https://github.com/jroimartin/gocui).

#### Exporting

You can use the `askgit export` sub command to save the output of queries into a sqlite database file.
The command expects a path to a db file (which will be created if it doesn't already exist) and a variable number of "export pairs," specified by the `-e` flag.
Each pair represents the name of a table to create and a query to generate its contents.

```
askgit export my-export-file -e commits,"SELECT * FROM commits" -e files,"SELECT * FROM files"
```

This can be useful if you're looking to use another tool to examine the data emitted by `askgit`.
Since the exported file is a plain SQLite database, queries should be much faster (as the original git repository is no longer traversed) and you should be able to use any tool that supports querying SQLite database files.
