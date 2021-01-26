package sync

import (
	"context"
	"fmt"
	"io"
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

func (cfg gitHubConfig) LookForRepos(console io.Writer) (
	idealRepoMap, rejectionReasonMap, error,
) {
	cred, err := git.FillCredential("https", cfg.Server)
	if err != nil {
		return nil, nil, err
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

	idealRepos := idealRepoMap{}
	rejectedRepos := rejectionReasonMap{}

	for owner := range cfg.Owners {
		fmt.Fprintf(console, "Finding repositories in %s/%s..", cfg.Server, owner)
		// TODO use repo type?
		rr, err := ghSAT.ReposInOrg(context.Background(), console, owner, github.AllRepos)
		if err != nil {
			fmt.Fprintf(console, " FAILED!\n")
			return nil, nil, err
		}
		fmt.Fprintf(console, " done.\n")

		for _, r := range rr {
			compURL, err := comparableRepoURL(r.CloneURL)
			if err != nil {
				return nil, nil, err
			}

			if reason := cfg.filterRepo(r); reason != "" {
				rejectedRepos[compURL] = reason
				continue
			}

			idealRepos[compURL] = idealRepo{
				Path:          cfg.pathForRepo(r.Owner.Login, r.Name),
				URL:           r.CloneURL,
				DefaultBranch: r.DefaultBranch,
			}
		}
	}

	return idealRepos, rejectedRepos, nil
}

func (cfg gitHubConfig) pathForRepo(owner, repo string) string {
	const defaultOwnerPrefixSeparator = "__"

	if cfg.SingleOwner != "" {
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
	ownerCriteria := cfg.Owners[repo.Owner.Login]

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
