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

const defaultOwnerPrefixSeparator = "__"

type gitHubConfig struct {
	Server string `yaml:"server"`

	SingleOrg string                                         `yaml:"org,omitempty"`
	Orgs      map[string]*gitHubConfigCriteriaWithExclusions `yaml:"orgs,omitempty"`

	SingleUser string                                         `yaml:"user,omitempty"`
	Users      map[string]*gitHubConfigCriteriaWithExclusions `yaml:"users,omitempty"`

	gitHubConfigCriteriaWithExclusions `yaml:",inline"`

	SingleDirForAllRepos *bool   `yaml:"singleDirForAllRepos,omitempty"`
	OwnerPrefixSeparator *string `yaml:"ownerPrefixSeparator,omitempty"`
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

func (cfg gitHubConfig) pathToOrgUserRepo(pathInsideRoot string) (org, user, repo string) {
	if pathInsideRoot == "" {
		panic("empty pathInsideRoot")
	}

	singleOwner := cfg.SingleOrg + cfg.SingleUser
	singleDir := singleOwner != ""
	if cfg.SingleDirForAllRepos != nil {
		singleDir = *cfg.SingleDirForAllRepos
	}

	parts := strings.SplitN(pathInsideRoot, string(filepath.Separator), 3)

	if singleDir {
		var prefixSeparator string
		if singleOwner == "" {
			prefixSeparator = defaultOwnerPrefixSeparator
		}
		if cfg.OwnerPrefixSeparator != nil {
			prefixSeparator = *cfg.OwnerPrefixSeparator
		}

		if prefixSeparator != "" {
			return "", "", strings.Replace(parts[0], prefixSeparator, "/", 1)
		}

		return "", "", fmt.Sprintf("%s/%s", singleOwner, parts[0])
	}

	if len(parts) > 1 {
		return "", "", fmt.Sprintf("%s/%s", parts[0], parts[1])
	}

	if _, ok := cfg.Orgs[parts[0]]; ok || cfg.SingleOrg == parts[0] {
		return parts[0], "", ""
	}

	if _, ok := cfg.Users[parts[0]]; ok || cfg.SingleUser == parts[0] {
		return "", parts[0], ""
	}

	return "", "", ""
}

// TODO revisit to simplify?
func (cfg gitHubConfig) pathForRepo(owner, repo string) string {

	if cfg.SingleOrg != "" {
		if cfg.SingleDirForAllRepos != nil && !*cfg.SingleDirForAllRepos {
			if cfg.OwnerPrefixSeparator != nil {
				return filepath.Join(
					owner, fmt.Sprintf("%s%s%s", owner, *cfg.OwnerPrefixSeparator, repo))
			}

			return filepath.Join(owner, repo)
		}

		if cfg.OwnerPrefixSeparator != nil {
			return fmt.Sprintf("%s%s%s", owner, *cfg.OwnerPrefixSeparator, repo)
		}

		return repo
	}

	if cfg.SingleDirForAllRepos != nil && *cfg.SingleDirForAllRepos {
		if cfg.OwnerPrefixSeparator != nil {
			return fmt.Sprintf("%s%s%s", owner, *cfg.OwnerPrefixSeparator, repo)
		}

		return fmt.Sprintf("%s%s%s", owner, defaultOwnerPrefixSeparator, repo)
	}

	if cfg.OwnerPrefixSeparator != nil {
		return filepath.Join(
			owner, fmt.Sprintf("%s%s%s", owner, *cfg.OwnerPrefixSeparator, repo))
	}

	return filepath.Join(owner, repo)
}

func (cfg gitHubConfig) filterRepo(repo github.Repo) string {
	var ownerCriteria *gitHubConfigCriteriaWithExclusions
	switch repo.Owner.Type {
	case "Organization":
		ownerCriteria = cfg.Orgs[repo.Owner.Login]
	case "User":
		ownerCriteria = cfg.Users[repo.Owner.Login]
	default:
		panic(fmt.Sprintf("unexpected repo.Owner.Type! repo = %#v", repo))
	}

	names := cfg.Names
	if ownerCriteria != nil && len(ownerCriteria.Names) > 0 {
		names = ownerCriteria.Names
	}
	if !matchesAnyFilter(repo.Name, names) {
		return fmt.Sprintf("%s doesn't match any name in %v", repo.Name, names)
	}

	topics := cfg.Topics
	if ownerCriteria != nil && len(ownerCriteria.Topics) > 0 {
		topics = ownerCriteria.Topics
	}
	if !anyWordMatchesAnyFilter(repo.Topics, topics) {
		return fmt.Sprintf("none of the repo topics %v match any config topic %v",
			repo.Topics, topics)
	}

	languages := cfg.Languages
	if ownerCriteria != nil && len(ownerCriteria.Languages) > 0 {
		languages = ownerCriteria.Languages
	}
	if !matchesAnyFilter(repo.Language, languages) {
		return fmt.Sprintf("%s doesn't match any language in %v", repo.Language, languages)
	}

	archived := cfg.Archived
	if ownerCriteria != nil && ownerCriteria.Archived != nil {
		archived = ownerCriteria.Archived
	}
	if archived != nil && repo.Archived != *archived {
		isOrIsNot := "is"
		if !repo.Archived {
			isOrIsNot = "is not"
		}
		return fmt.Sprintf("repo %s archived", isOrIsNot)
	}

	fork := cfg.Fork
	if ownerCriteria != nil && ownerCriteria.Fork != nil {
		fork = ownerCriteria.Fork
	}
	if fork != nil && repo.Fork != *fork {
		isOrIsNot := "is"
		if !repo.Fork {
			isOrIsNot = "is not"
		}
		return fmt.Sprintf("repo %s a fork", isOrIsNot)
	}

	isTemplate := cfg.IsTemplate
	if ownerCriteria != nil && ownerCriteria.IsTemplate != nil {
		isTemplate = ownerCriteria.IsTemplate
	}
	if isTemplate != nil && repo.IsTemplate != *isTemplate {
		isOrIsNot := "is"
		if !repo.IsTemplate {
			isOrIsNot = "is not"
		}
		return fmt.Sprintf("repo %s a template", isOrIsNot)
	}

	private := cfg.Private
	if ownerCriteria != nil && ownerCriteria.Private != nil {
		private = ownerCriteria.Private
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
			github.RepoOwner{Login: orgName, Type: "Organization"},
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
			github.RepoOwner{Login: userName, Type: "User"},
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
			filepath.Join(syncRoot, cfg.pathForRepo(r.Owner.Login, r.Name)))
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
		owner := strings.SplitN(repo, string(filepath.Separator), 2)[0]
		if !orgsSet[owner] && !usersSet[owner] {
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
