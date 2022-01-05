package github

import (
	"context"
	"time"

	"github.com/mergestat/mergestat/extensions/options"
	"github.com/pkg/errors"
	"github.com/rs/zerolog"
	"github.com/shurcooL/githubv4"
	"go.riyazali.net/sqlite"
	"golang.org/x/oauth2"
	"golang.org/x/time/rate"
)

// Register registers GitHub related functionality as a SQLite extension
func Register(ext *sqlite.ExtensionApi, opt *options.Options) (_ sqlite.ErrorCode, err error) {
	rateLimiter := GetGitHubRateLimitFromCtx(opt.Context)
	if rateLimiter == nil {
		rateLimiter = rate.NewLimiter(rate.Every(1*time.Second), 2)
	}

	if opt.Logger == nil {
		l := zerolog.Nop()
		opt.Logger = &l
	}

	githubOpts := &Options{
		RateLimiter: rateLimiter,
		Client: func() *githubv4.Client {
			httpClient := oauth2.NewClient(context.Background(), oauth2.StaticTokenSource(
				&oauth2.Token{AccessToken: GetGitHubTokenFromCtx(opt.Context)},
			))
			client := githubv4.NewClient(httpClient)
			return client
		},
		PerPage: GetGitHubPerPageFromCtx(opt.Context),
		Logger:  opt.Logger,
	}

	if opt.GitHubClientGetter != nil {
		githubOpts.Client = opt.GitHubClientGetter
	}

	var modules = map[string]sqlite.Module{
		"github_stargazers":              NewStargazersModule(githubOpts),
		"github_starred_repos":           NewStarredReposModule(githubOpts),
		"github_user_repos":              NewUserReposModule(githubOpts),
		"github_org_repos":               NewOrgReposModule(githubOpts),
		"github_repo_issues":             NewIssuesModule(githubOpts),
		"github_repo_pull_requests":      NewPRModule(githubOpts),
		"github_repo_branch_protections": NewProtectionsModule(githubOpts),
		"github_repo_issue_comments":     NewIssueCommentsModule(githubOpts),
		"github_repo_pr_comments":        NewPRCommentsModule(githubOpts),
		"github_repo_branches":           NewBranchModule(githubOpts),
		"github_repo_pr_commits":         NewPRCommitsModule(githubOpts),
	}

	modules["github_issue_comments"] = modules["github_repo_issue_comments"]
	modules["github_pr_comments"] = modules["github_repo_pr_comments"]
	modules["github_issues"] = modules["github_repo_issues"]
	modules["github_pull_requests"] = modules["github_repo_pull_requests"]
	modules["github_prs"] = modules["github_repo_pull_requests"]
	modules["github_repo_prs"] = modules["github_repo_pull_requests"]
	modules["github_branch_protections"] = modules["github_repo_branch_protections"]
	modules["github_pr_commits"] = modules["github_repo_pr_commits"]

	// register GitHub tables
	for name, mod := range modules {
		if err = ext.CreateModule(name, mod); err != nil {
			return sqlite.SQLITE_ERROR, errors.Wrapf(err, "failed to register GitHub %q module", name)
		}
	}

	var fns = map[string]sqlite.Function{
		"github_stargazer_count":   NewStarredReposFunc(githubOpts),
		"github_repo_file_content": NewRepoFileContentFunc(githubOpts),
		"github_user":              NewGitHubUserFunc(githubOpts),
	}

	// register GitHub funcs
	for name, fn := range fns {
		if err = ext.CreateFunction(name, fn); err != nil {
			return sqlite.SQLITE_ERROR, errors.Wrapf(err, "failed to register GitHub %q function", name)
		}
	}
	return sqlite.SQLITE_OK, nil
}
