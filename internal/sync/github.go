package sync

import (
	"context"
	"fmt"
	"io"
	"path/filepath"
	"strings"

	"github.com/mikesep/frond/internal/git"
	"github.com/mikesep/frond/internal/github"
)

const defaultAccountPrefixSeparator = "__"

type gitHubConfig struct {
	Server string `yaml:"server"`

	SingleOrg string                                         `yaml:"org,omitempty"`
	Orgs      map[string]*gitHubConfigCriteriaWithExclusions `yaml:"orgs,omitempty"`

	SingleUser string                                         `yaml:"user,omitempty"`
	Users      map[string]*gitHubConfigCriteriaWithExclusions `yaml:"users,omitempty"`

	gitHubConfigCriteriaWithExclusions `yaml:",inline"`

	SingleDirForAllRepos   *bool   `yaml:"singleDirForAllRepos,omitempty"`
	AccountPrefixSeparator *string `yaml:"accountPrefixSeparator,omitempty"`
}

type gitHubConfigCriteriaWithExclusions struct {
	gitHubConfigCriteria `yaml:",inline"`
	Exclude              *gitHubConfigCriteria `yaml:"exclude,omitempty"`
}

type gitHubConfigCriteria struct {
	Names []string `yaml:"names,omitempty"`

	Topics    []string `yaml:"topics,omitempty"`
	Languages []string `yaml:"languages,omitempty"`

	Archived   *bool `yaml:"archived,omitempty"`
	Fork       *bool `yaml:"fork,omitempty"`
	IsTemplate *bool `yaml:"is_template,omitempty"`
	Private    *bool `yaml:"private,omitempty"`
}

//------------------------------------------------------------------------------

func (cfg gitHubConfig) validate() error {
	if cfg.Server == "" {
		return fmt.Errorf("server is missing")
	}

	if cfg.SingleOrg != "" {
		switch {
		case len(cfg.Orgs) != 0:
			return fmt.Errorf("cannot have org and orgs")
		case cfg.SingleUser != "":
			return fmt.Errorf("cannot have org and user")
		case len(cfg.Users) != 0:
			return fmt.Errorf("cannot have org and users")
		}
	}

	if cfg.SingleUser != "" {
		switch {
		case cfg.SingleOrg != "":
			return fmt.Errorf("cannot have user and org")
		case len(cfg.Orgs) != 0:
			return fmt.Errorf("cannot have user and orgs")
		case len(cfg.Users) != 0:
			return fmt.Errorf("cannot have user and users")
		}
	}

	return nil
}

func (cfg gitHubConfig) pathToOrgUserRepo(pathInsideRoot string) (org, user, repo string) {
	if pathInsideRoot == "" {
		panic("empty pathInsideRoot")
	}

	singleAccount := (cfg.SingleOrg + cfg.SingleUser) != ""
	singleDir := singleAccount
	if cfg.SingleDirForAllRepos != nil {
		singleDir = *cfg.SingleDirForAllRepos
	}

	parts := strings.SplitN(pathInsideRoot, string(filepath.Separator), 3)

	if singleDir {
		var prefixSeparator string
		if !singleAccount {
			prefixSeparator = defaultAccountPrefixSeparator
		}
		if cfg.AccountPrefixSeparator != nil {
			prefixSeparator = *cfg.AccountPrefixSeparator
		}

		if prefixSeparator == "" {
			return cfg.SingleOrg, cfg.SingleUser, parts[0]
		}

		separated := strings.SplitN(parts[0], prefixSeparator, 2)

		org, user := cfg.orgOrUser(separated[0])
		if org == "" && user == "" {
			return "", "", ""
		}

		return org, user, separated[1]
	}

	org, user = cfg.orgOrUser(parts[0])
	if org == "" && user == "" {
		return "", "", ""
	}

	if len(parts) < 2 {
		return org, user, ""
	}

	var prefixSeparator string
	if cfg.AccountPrefixSeparator != nil {
		prefixSeparator = *cfg.AccountPrefixSeparator
	}

	if prefixSeparator == "" {
		return org, user, parts[1]
	}

	separated := strings.SplitN(parts[1], prefixSeparator, 2)
	if separated[0] != org+user {
		panic("strange, dir doesn't match prefix: " + pathInsideRoot)
	}

	return org, user, separated[1]
}

func (cfg gitHubConfig) pathForRepo(account, repo string) string {
	if org, user := cfg.orgOrUser(account); org == "" && user == "" {
		panic(fmt.Sprintf("pathForRepo: account %q is neither an org or user", account))
	}

	singleAccount := cfg.SingleOrg + cfg.SingleUser
	singleDir := singleAccount != ""
	if cfg.SingleDirForAllRepos != nil {
		singleDir = *cfg.SingleDirForAllRepos
	}

	if singleDir {
		var prefixSeparator string
		if singleAccount == "" {
			prefixSeparator = defaultAccountPrefixSeparator
		}
		if cfg.AccountPrefixSeparator != nil {
			prefixSeparator = *cfg.AccountPrefixSeparator
		}

		if prefixSeparator == "" {
			return repo
		}

		return fmt.Sprintf("%s%s%s", account, prefixSeparator, repo)
	}

	repoDir := repo
	if cfg.AccountPrefixSeparator != nil {
		repoDir = fmt.Sprintf("%s%s%s", account, *cfg.AccountPrefixSeparator, repo)
	}

	return filepath.Join(account, repoDir)
}

func (cfg gitHubConfig) filterRepo(repo github.Repo) string {
	var accountCriteria *gitHubConfigCriteriaWithExclusions
	switch repo.Account.Type {
	case "Organization":
		accountCriteria = cfg.Orgs[repo.Account.Login]
	case "User":
		accountCriteria = cfg.Users[repo.Account.Login]
	default:
		panic(fmt.Sprintf("unexpected repo.Account.Type! repo = %#v", repo))
	}

	names := cfg.Names
	if accountCriteria != nil && len(accountCriteria.Names) > 0 {
		names = accountCriteria.Names
	}
	if !matchesAnyFilter(repo.Name, names) {
		return fmt.Sprintf("%s doesn't match any name in %v", repo.Name, names)
	}

	topics := cfg.Topics
	if accountCriteria != nil && len(accountCriteria.Topics) > 0 {
		topics = accountCriteria.Topics
	}
	if !anyWordMatchesAnyFilter(repo.Topics, topics) {
		return fmt.Sprintf("none of the repo topics %v match any config topic %v",
			repo.Topics, topics)
	}

	languages := cfg.Languages
	if accountCriteria != nil && len(accountCriteria.Languages) > 0 {
		languages = accountCriteria.Languages
	}
	if !matchesAnyFilter(repo.Language, languages) {
		return fmt.Sprintf("%s doesn't match any language in %v", repo.Language, languages)
	}

	archived := cfg.Archived
	if accountCriteria != nil && accountCriteria.Archived != nil {
		archived = accountCriteria.Archived
	}
	if archived != nil && repo.Archived != *archived {
		isOrIsNot := "is"
		if !repo.Archived {
			isOrIsNot = "is not"
		}
		return fmt.Sprintf("repo %s archived", isOrIsNot)
	}

	fork := cfg.Fork
	if accountCriteria != nil && accountCriteria.Fork != nil {
		fork = accountCriteria.Fork
	}
	if fork != nil && repo.Fork != *fork {
		isOrIsNot := "is"
		if !repo.Fork {
			isOrIsNot = "is not"
		}
		return fmt.Sprintf("repo %s a fork", isOrIsNot)
	}

	isTemplate := cfg.IsTemplate
	if accountCriteria != nil && accountCriteria.IsTemplate != nil {
		isTemplate = accountCriteria.IsTemplate
	}
	if isTemplate != nil && repo.IsTemplate != *isTemplate {
		isOrIsNot := "is"
		if !repo.IsTemplate {
			isOrIsNot = "is not"
		}
		return fmt.Sprintf("repo %s a template", isOrIsNot)
	}

	private := cfg.Private
	if accountCriteria != nil && accountCriteria.Private != nil {
		private = accountCriteria.Private
	}
	if private != nil && repo.Private != *private {
		isOrIsNot := "is"
		if !repo.Private {
			isOrIsNot = "is not"
		}
		return fmt.Sprintf("repo %s private", isOrIsNot)
	}

	return ""
}

func (cfg gitHubConfig) orgOrUser(name string) (org, user string) {
	if _, ok := cfg.Orgs[name]; ok || cfg.SingleOrg == name {
		return name, ""
	}

	if _, ok := cfg.Users[name]; ok || cfg.SingleUser == name {
		return "", name
	}

	return "", ""
}

//------------------------------------------------------------------------------

func findGitHubRepos(
	syncRoot, workDir string, cmdArgs []string, cfg *gitHubConfig, console io.Writer,
) (idealRepoMap, rejectionReasonMap, error) {
	ctx := context.Background()

	orgs, users, individualRepos, err := cmdlineToGitHubOrgsUsersRepos(syncRoot, workDir, cmdArgs, cfg)
	if err != nil {
		return nil, nil, err
	}

	cred, err := git.FillCredential("https", cfg.Server)
	if err != nil {
		return nil, nil, err
	}
	ghSAT := github.ServerAndToken{
		Server: cfg.Server,
		Token:  cred.Password,
	}

	var unfilteredRepos []github.Repo

	// TODO use repo type instead of github.AllRepos when appropriate?

	for _, orgName := range orgs {
		fmt.Fprintf(console, "Finding repositories in %s/%s..", cfg.Server, orgName)
		rr, err := ghSAT.ListRepos(ctx, console,
			github.Account{Login: orgName, Type: "Organization"},
			github.AllRepos)
		if err != nil {
			fmt.Fprintf(console, " FAILED!\n")
			return nil, nil, err
		}
		fmt.Fprintf(console, " done.\n")

		unfilteredRepos = append(unfilteredRepos, rr...)
	}

	for _, userName := range users {
		fmt.Fprintf(console, "Finding repositories in %s/%s..", cfg.Server, userName)
		rr, err := ghSAT.ListRepos(ctx, console,
			github.Account{Login: userName, Type: "User"},
			github.AllRepos)
		if err != nil {
			fmt.Fprintf(console, " FAILED!\n")
			return nil, nil, err
		}
		fmt.Fprintf(console, " done.\n")

		unfilteredRepos = append(unfilteredRepos, rr...)
	}

	if len(individualRepos) > 0 {
		fmt.Fprintf(console, "Finding individual repositories..")
	}
	for _, repoName := range individualRepos {
		fmt.Fprint(console, ".")
		r, err := ghSAT.GetRepo(ctx, repoName)
		if err != nil {
			fmt.Fprintf(console, " FAILED!\n")
			return nil, nil, err
		}
		unfilteredRepos = append(unfilteredRepos, r)
	}
	if len(individualRepos) > 0 {
		fmt.Fprintf(console, " done.\n")
	}

	// filter the repos

	idealRepos := idealRepoMap{}
	rejectedRepos := rejectionReasonMap{}

	for _, r := range unfilteredRepos {
		compURL, err := comparableRepoURL(r.CloneURL)
		if err != nil {
			return nil, nil, err
		}

		if reason := cfg.filterRepo(r); reason != "" {
			rejectedRepos[compURL] = reason
			continue
		}

		pathToRepo, err := filepath.Rel(workDir,
			filepath.Join(syncRoot, cfg.pathForRepo(r.Account.Login, r.Name)))
		if err != nil {
			return nil, nil, err
		}

		idealRepos[compURL] = idealRepo{
			Path:          pathToRepo,
			URL:           r.CloneURL,
			DefaultBranch: r.DefaultBranch,
		}
	}

	return idealRepos, rejectedRepos, nil
}

func cmdlineToGitHubOrgsUsersRepos(syncRoot, workDir string, cmdArgs []string, cfg *gitHubConfig,
) (orgs, users, repos []string, err error) {
	if !filepath.IsAbs(syncRoot) {
		panic(fmt.Sprintf("syncRoot is not an absolute path: %q", syncRoot))
	}
	if !filepath.IsAbs(workDir) {
		panic(fmt.Sprintf("workDir is not an absolute path: %q", workDir))
	}

	var syncAll bool

	if len(cmdArgs) == 0 {
		syncAll = true
	}

	for _, arg := range cmdArgs {
		argPath := arg
		if !filepath.IsAbs(arg) {
			argPath = filepath.Join(workDir, arg)
		}

		if argPath == syncRoot {
			syncAll = true
			break
		}
	}

	if syncAll {
		if cfg.SingleOrg != "" {
			orgs = append(orgs, cfg.SingleOrg)
		} else {
			orgs = make([]string, 0, len(cfg.Orgs))
			for name := range cfg.Orgs {
				orgs = append(orgs, name)
			}
		}

		if cfg.SingleUser != "" {
			users = append(users, cfg.SingleUser)
		} else {
			users = make([]string, 0, len(cfg.Users))
			for name := range cfg.Users {
				users = append(users, name)
			}
		}

		return orgs, users, nil, nil
	}

	orgsSet := map[string]bool{}
	usersSet := map[string]bool{}
	reposSet := map[string]bool{}

	for _, arg := range cmdArgs {
		argPath := arg
		if !filepath.IsAbs(arg) {
			argPath = filepath.Join(workDir, arg)
		}

		relToRoot, err := filepath.Rel(syncRoot, argPath)
		if err != nil {
			return nil, nil, nil, err
		}

		if strings.HasPrefix(relToRoot, "..") {
			return nil, nil, nil, fmt.Errorf("arg %q points outside sync root %q", arg, syncRoot)
		}

		org, user, repo := cfg.pathToOrgUserRepo(relToRoot)
		switch {
		case org != "":
			orgsSet[org] = true
		case user != "":
			usersSet[user] = true
		case repo != "":
			reposSet[repo] = true
		default:
			panic(fmt.Sprintf("all blanks returned from pathToOrgUserRepo for %q", relToRoot))
		}
	}

	for repo := range reposSet {
		account := strings.SplitN(repo, string(filepath.Separator), 2)[0]
		if !orgsSet[account] && !usersSet[account] {
			repos = append(repos, repo)
		}
	}

	for name := range orgsSet {
		orgs = append(orgs, name)
	}

	for name := range usersSet {
		users = append(users, name)
	}

	return orgs, users, repos, nil
}
