package sync

import (
	"context"
	"fmt"
	"path/filepath"

	"github.com/mikesep/frond/internal/git"
	"github.com/mikesep/frond/internal/github"
)

type gitHubConfig struct {
	Server string `yaml:"server"`

	SingleOwner string                                         `yaml:"owner,omitempty"`
	Owners      map[string]*gitHubConfigCriteriaWithExclusions `yaml:"owners,omitempty"`

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

func (cfg gitHubConfig) getReposAtPaths() ([]repoAtPath, error) {
	cred, err := git.FillCredential("https", cfg.Server)
	if err != nil {
		return nil, err
	}

	ghSAT := github.ServerAndToken{
		Server: cfg.Server,
		Token:  cred.Password,
	}

	if cfg.SingleOwner != "" {
		cfg.Owners = map[string]*gitHubConfigCriteriaWithExclusions{
			cfg.SingleOwner: nil,
		}
	}

	filter := func(r github.Repo) bool {
		return filterGitHubRepo(cfg, r)
	}
	// TODO use repo type?

	var repos []repoAtPath

	for owner := range cfg.Owners {
		rr, err := ghSAT.ReposInOrg(context.Background(), owner, github.AllRepos, filter)
		if err != nil {
			return nil, err
		}
		for _, r := range rr {
			repos = append(repos, repoAtPath{
				Path: cfg.pathForRepo(r.Owner.Login, r.Name),
				URL:  r.CloneURL,
			})
		}
	}

	return repos, nil
}

func (cfg gitHubConfig) pathForRepo(owner, repo string) string {
	const defaultOwnerPrefixSeparator = "__"

	if cfg.SingleOwner != "" {
		if cfg.SingleDirForAllRepos != nil && *cfg.SingleDirForAllRepos == false {
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

	if cfg.SingleDirForAllRepos != nil && *cfg.SingleDirForAllRepos == true {
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

func filterGitHubRepo(cfg gitHubConfig, repo github.Repo) bool {
	ownerCriteria := cfg.Owners[repo.Owner.Login]

	names := cfg.Names
	if ownerCriteria != nil && len(ownerCriteria.Names) > 0 {
		names = ownerCriteria.Names
	}
	if !matchesAnyFilter(repo.Name, names) {
		// fmt.Printf("%v doesn't match any names in (%v)\n", repo.Name, names)
		return false
	}

	topics := cfg.Topics
	if ownerCriteria != nil && len(ownerCriteria.Topics) > 0 {
		topics = ownerCriteria.Topics
	}
	if !anyWordMatchesAnyFilter(repo.Topics, topics) {
		// fmt.Printf("%v: repo.Topics (%v) doesn't match any topics in (%v)\n", repo.FullName, repo.Topics, topics)
		return false
	}

	languages := cfg.Languages
	if ownerCriteria != nil && len(ownerCriteria.Languages) > 0 {
		topics = ownerCriteria.Languages
	}
	if !matchesAnyFilter(repo.Language, languages) {
		// fmt.Printf("%v: %v doesn't match any languages in (%v)\n", repo.FullName, repo.Language, languages)
		return false
	}

	archived := cfg.Archived
	if ownerCriteria != nil && ownerCriteria.Archived != nil {
		archived = ownerCriteria.Archived
	}
	if archived != nil && repo.Archived != *archived {
		return false
	}

	fork := cfg.Fork
	if ownerCriteria != nil && ownerCriteria.Fork != nil {
		fork = ownerCriteria.Fork
	}
	if fork != nil && repo.Fork != *fork {
		return false
	}

	isTemplate := cfg.IsTemplate
	if ownerCriteria != nil && ownerCriteria.IsTemplate != nil {
		isTemplate = ownerCriteria.IsTemplate
	}
	if isTemplate != nil && repo.IsTemplate != *isTemplate {
		return false
	}

	private := cfg.Private
	if ownerCriteria != nil && ownerCriteria.Private != nil {
		private = ownerCriteria.Private
	}
	if private != nil && repo.Private != *private {
		return false
	}

	return true
}
