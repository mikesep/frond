// SPDX-FileCopyrightText: 2020 Michael Seplowitz
// SPDX-License-Identifier: MIT

package github

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
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
	Name     string  `json:"name"`
	FullName string  `json:"full_name"`
	Account  Account `json:"owner"`
	CloneURL string  `json:"clone_url"`

	Archived      bool     `json:"archived"`
	DefaultBranch string   `json:"default_branch"`
	Fork          bool     `json:"fork"`
	IsTemplate    bool     `json:"is_template"`
	Language      string   `json:"language"`
	Private       bool     `json:"private"`
	Topics        []string `json:"topics"`
}

type Account struct {
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

func (sat ServerAndToken) GetRepo(ctx context.Context, fullName string) (Repo, error) {
	apiURL := fmt.Sprintf("%s/repos/%s", sat.restV3URL(), url.PathEscape(fullName))

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, apiURL, nil)
	if err != nil {
		return Repo{}, err
	}

	req.Header.Set("Authorization", "token "+sat.Token)
	req.Header.Set("Accept", "application/vnd.github.v3+json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return Repo{}, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return Repo{}, fmt.Errorf("bad status code %d from %s", resp.StatusCode, apiURL)
	}

	respBytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return Repo{}, err
	}

	var repoInfo Repo
	if err := json.Unmarshal(respBytes, &repoInfo); err != nil {
		return Repo{}, err
	}

	return repoInfo, nil
}

func (sat ServerAndToken) ListRepos(
	ctx context.Context, progress io.Writer, account Account, repoType RepoType,
) ([]Repo, error) {
	var results []Repo

	orgsOrUsers := "orgs"
	if account.Type == "User" {
		orgsOrUsers = "users"
	}

	nextURL := fmt.Sprintf("%s/%s/%s/repos?type=%s&per_page=100&sort=full_name",
		sat.restV3URL(), orgsOrUsers, account.Login, repoType)

	for nextURL != "" {
		fmt.Fprint(progress, ".") // print a dot for each iteration

		req, err := http.NewRequestWithContext(ctx, http.MethodGet, nextURL, nil)
		if err != nil {
			return results, err
		}

		req.Header.Set("Authorization", "token "+sat.Token)
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

		results = append(results, repos...)

		nextURL = getRelFromLinkHeader(resp.Header.Get("Link"), "next")
	}

	return results, nil
}

func (sat ServerAndToken) GetAccount(ctx context.Context, name string) (Account, error) {
	var acct Account

	queryURL := fmt.Sprintf("%s/users/%s", sat.restV3URL(), url.PathEscape(name))

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, queryURL, nil)
	if err != nil {
		return acct, err
	}

	req.Header.Set("Authorization", "token "+sat.Token)
	req.Header.Set("Accept", "application/vnd.github.v3+json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return acct, err
	}
	defer resp.Body.Close()

	switch resp.StatusCode {
	case http.StatusOK:
		// yay!
	case http.StatusNotFound:
		return acct, fmt.Errorf("failed to find account %q", name)
	default:
		return acct, fmt.Errorf("bad status code: %v", resp.StatusCode)
	}

	respBytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return acct, err
	}

	err = json.Unmarshal(respBytes, &acct)
	return acct, err
}

//------------------------------------------------------------------------------

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
