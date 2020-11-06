package github

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
)

func (sat *ServerAndToken) restV3URL() string {
	if sat.Server == "github.com" {
		return "https://api.github.com"
	}

	return "https://" + sat.Server + "/api/v3"
}

//------------------------------------------------------------------------------

type Repo struct {
	Name     string    `json:"name"`
	FullName string    `json:"full_name"`
	Owner    RepoOwner `json:"owner"`

	Archived      bool     `json:"archived"`
	DefaultBranch string   `json:"default_branch"`
	Fork          bool     `json:"fork"`
	IsTemplate    bool     `json:"is_template"`
	Language      string   `json:"language"`
	Private       bool     `json:"private"`
	Topics        []string `json:"topics"`
}

type RepoOwner struct {
	Login string `json:"login"`
	Type  string `json:"type"` // User or Organization
}

type RepoType string

const (
	AllRepos      RepoType = "all"
	PublicRepos            = "public"
	PrivateRepos           = "private"
	ForkRepos              = "forks"
	SourceRepos            = "sources"
	MemberRepos            = "member"
	InternalRepos          = "internal"
)

type RepoFilterFunc func(r Repo) bool

func (sat ServerAndToken) ReposInOrg(
	ctx context.Context, org string, repoType RepoType, repoFilterFunc RepoFilterFunc,
) ([]Repo, error) {
	var results []Repo

	nextURL := fmt.Sprintf("%s/orgs/%s/repos?type=%s&per_page=100&sort=full_name",
		sat.restV3URL(), org, repoType)

	for {
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, nextURL, nil)
		if err != nil {
			return results, err
		}

		req.Header.Set("Authorization", "bearer "+sat.Token)
		req.Header.Set("Accept", "application/vnd.github.v3+json")

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			return results, err
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			return results, fmt.Errorf("bad status code: %v", resp.StatusCode)
		}

		respBytes, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return results, err
		}

		var repos []Repo
		if err := json.Unmarshal(respBytes, &repos); err != nil {
			return nil, err
		}

		if repoFilterFunc != nil {
			for _, r := range repos {
				if repoFilterFunc(r) {
					results = append(results, r)
				}
			}
		} else {
			results = append(results, repos...)
		}

		nextURL = getRelFromLinkHeader(resp.Header.Get("Link"), "next")
		if nextURL == "" {
			break
		}
	}

	return results, nil
}

func getRelFromLinkHeader(header, page string) string {
	if header == "" {
		return ""
	}

	rel := fmt.Sprintf(`rel="%s"`, page)
	links := strings.Split(header, ", ")
	for _, l := range links {
		parts := strings.Split(l, "; ")
		if parts[1] == rel {
			return strings.Trim(parts[0], "<>")
		}
	}
	return ""
}
