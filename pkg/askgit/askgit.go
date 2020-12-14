package askgit

import (
	"crypto/md5"
	"database/sql"
	"fmt"
	"os"
	"os/exec"
	"os/user"
	"path"
	"strings"
	"time"

	"github.com/augmentable-dev/askgit/pkg/ghqlite"
	"github.com/augmentable-dev/askgit/pkg/gitqlite"
	"github.com/gitsight/go-vcsurl"
	git "github.com/libgit2/git2go/v31"
	"github.com/mattn/go-sqlite3"
	"golang.org/x/time/rate"
)

func init() {
	sql.Register("askgit", &sqlite3.SQLiteDriver{
		ConnectHook: func(conn *sqlite3.SQLiteConn) error {
			err := conn.CreateModule("git_log", gitqlite.NewGitLogModule())
			if err != nil {
				return err
			}

			err = conn.CreateModule("git_log_cli", gitqlite.NewGitLogCLIModule())
			if err != nil {
				return err
			}

			err = conn.CreateModule("git_tree", gitqlite.NewGitFilesModule())
			if err != nil {
				return err
			}

			err = conn.CreateModule("git_tag", gitqlite.NewGitTagsModule())
			if err != nil {
				return err
			}

			err = conn.CreateModule("git_branch", gitqlite.NewGitBranchesModule())
			if err != nil {
				return err
			}

			err = conn.CreateModule("git_stats", gitqlite.NewGitStatsModule())
			if err != nil {
				return err
			}

			githubToken := os.Getenv("GITHUB_TOKEN")
			rateLimiter := rate.NewLimiter(rate.Every(2*time.Second), 1)

			err = conn.CreateModule("github_org_repos", ghqlite.NewReposModule(ghqlite.OwnerTypeOrganization, ghqlite.ReposModuleOptions{
				Token:       githubToken,
				RateLimiter: rateLimiter,
			}))
			if err != nil {
				return err
			}

			err = conn.CreateModule("github_user_repos", ghqlite.NewReposModule(ghqlite.OwnerTypeUser, ghqlite.ReposModuleOptions{
				Token:       githubToken,
				RateLimiter: rateLimiter,
			}))
			if err != nil {
				return err
			}

			err = conn.CreateModule("github_pull_requests", ghqlite.NewPullRequestsModule(ghqlite.PullRequestsModuleOptions{
				Token:       githubToken,
				RateLimiter: rateLimiter,
			}))
			if err != nil {
				return err
			}

			err = loadHelperFuncs(conn)
			if err != nil {
				return err
			}

			return nil
		},
	})
}

type AskGit struct {
	db       *sql.DB
	repoPath string
	options  *Options
}

type Options struct {
	UseGitCLI   bool
	GitHubToken string
}

// New creates an instance of AskGit
func New(repoPath string, options *Options) (*AskGit, error) {
	// TODO with the addition of the GitHub API virtual tables, repoPath should no longer be required for creating
	// as *AskGit instance, as the caller may just be interested in querying against the GitHub API (or some other
	// to be define virtual table that doesn't need a repo on disk).
	// This should be reformulated, as it means currently the askgit command requires a local git repo, even if the query
	// only executes agains the GitHub API

	// see https://github.com/mattn/go-sqlite3/issues/204
	// also mentioned in the FAQ of the README: https://github.com/mattn/go-sqlite3#faq
	db, err := sql.Open("askgit", fmt.Sprintf("file:%x?mode=memory&cache=shared", md5.Sum([]byte(repoPath))))
	if err != nil {
		return nil, err
	}
	_, err = git.OpenRepository(repoPath)
	if err != nil {
		return nil, err
	}

	g := &AskGit{db: db, repoPath: repoPath, options: options}

	err = g.ensureTables(options)
	if err != nil {
		return nil, err
	}
	return g, nil
}

func (a *AskGit) DB() *sql.DB {
	return a.db
}

func (a *AskGit) RepoPath() string {
	return a.repoPath
}

// creates the virtual tables inside of the *sql.DB
func (a *AskGit) ensureTables(options *Options) error {
	_, err := exec.LookPath("git")
	localGitExists := err == nil
	a.repoPath = strings.ReplaceAll(a.repoPath, "'", "''")
	if !options.UseGitCLI || !localGitExists {
		_, err := a.db.Exec(fmt.Sprintf("CREATE VIRTUAL TABLE IF NOT EXISTS commits USING git_log('%s');", a.repoPath))
		if err != nil {
			return err
		}

	} else {
		_, err := a.db.Exec(fmt.Sprintf("CREATE VIRTUAL TABLE IF NOT EXISTS commits USING git_log_cli('%s');", a.repoPath))
		if err != nil {
			return err
		}

	}
	_, err = a.db.Exec(fmt.Sprintf("CREATE VIRTUAL TABLE IF NOT EXISTS stats USING git_stats('%s');", a.repoPath))
	if err != nil {
		return err
	}

	_, err = a.db.Exec(fmt.Sprintf("CREATE VIRTUAL TABLE IF NOT EXISTS files USING git_tree('%s');", a.repoPath))
	if err != nil {
		return err
	}
	_, err = a.db.Exec(fmt.Sprintf("CREATE VIRTUAL TABLE IF NOT EXISTS tags USING git_tag('%s');", a.repoPath))
	if err != nil {
		return err
	}
	_, err = a.db.Exec(fmt.Sprintf("CREATE VIRTUAL TABLE IF NOT EXISTS branches USING git_branch('%s');", a.repoPath))
	if err != nil {
		return err
	}

	return nil
}

func CreateAuthenticationCallback(remote *vcsurl.VCS) *git.CloneOptions {
	cloneOptions := &git.CloneOptions{}

	if _, err := remote.Remote(vcsurl.SSH); err == nil { // if SSH, use "default" credentials
		// use FetchOptions instead of directly RemoteCallbacks
		// https://github.com/libgit2/git2go/commit/36e0a256fe79f87447bb730fda53e5cbc90eb47c
		cloneOptions.FetchOptions = &git.FetchOptions{
			RemoteCallbacks: git.RemoteCallbacks{
				CredentialsCallback: func(url string, username string, allowedTypes git.CredType) (*git.Cred, error) {
					usr, _ := user.Current()
					publicSSH := path.Join(usr.HomeDir, ".ssh/id_rsa.pub")
					privateSSH := path.Join(usr.HomeDir, ".ssh/id_rsa")

					cred, ret := git.NewCredSshKey("git", publicSSH, privateSSH, "")
					return cred, ret
				},
				CertificateCheckCallback: func(cert *git.Certificate, valid bool, hostname string) git.ErrorCode {
					return git.ErrOk
				},
			}}
	}
	return cloneOptions
}
