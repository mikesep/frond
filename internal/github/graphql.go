package github

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
)

func graphQLQuery() {
}

type ServerAndToken struct {
	Server string // github.com or ghes.example.com
	Token  string
}

func (sat *ServerAndToken) apiURL() string {
	if sat.Server == "github.com" {
		return "https://api.github.com"
	}

	return "https://" + sat.Server + "/api"
}

func (sat *ServerAndToken) graphQuery(query string, variables interface{}, data interface{}) error {
	bodyObj := struct {
		Query     string      `json:"query"`
		Variables interface{} `json:"variables"`
	}{
		Query:     query,
		Variables: variables,
	}

	body, err := json.Marshal(&bodyObj)
	if err != nil {
		return err
	}

	req, err := http.NewRequest(http.MethodPost, sat.apiURL()+"/graphql", bytes.NewReader(body))
	if err != nil {
		return err
	}

	req.Header.Set("Authorization", "bearer "+sat.Token)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	respBytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	type graphQLError struct {
		Message string `json:"message"`
	}

	type dataOrError struct {
		Data   *json.RawMessage `json:"data"`
		Errors *[]graphQLError  `json:"errors"`
	}

	var doe dataOrError

	if err := json.Unmarshal(respBytes, &doe); err != nil {
		return err
	}

	if doe.Errors != nil {
		return fmt.Errorf("%+v", *doe.Errors)
	}

	return json.Unmarshal(*doe.Data, &data)
}

func (sat *ServerAndToken) OrgRepos(org string) ([]string, error) {
	const query = `
      query($org: String!, $prevEndCursor: String) {
        organization(login: $org) {
		  repositories(first: 100, after: $prevEndCursor) {
            nodes {
              name
			  isArchived
            }
            pageInfo {
              endCursor
              hasNextPage
            }
          }
        }
      }`

	vars := map[string]string{
		"org": org,
	}

	type node struct {
		Name       string `json:"name"`
		IsArchived bool   `json:"isArchived"`
	}

	type pageInfo struct {
		EndCursor   string `json:"endCursor"`
		HasNextPage bool   `json:"hasNextPage"`
	}

	type repositories struct {
		Nodes    []node   `json:"nodes"`
		PageInfo pageInfo `json:"pageInfo"`
	}

	type organization struct {
		Repos repositories `json:"repositories"`
	}

	type data struct {
		Org organization `json:"organization"`
	}

	var repos []string

	for {
		var data data

		if err := sat.graphQuery(query, vars, &data); err != nil {
			return nil, err
		}

		for _, r := range data.Org.Repos.Nodes {
			if !r.IsArchived { // TODO make an option?
				repos = append(repos, r.Name)
			}
		}

		if !data.Org.Repos.PageInfo.HasNextPage {
			return repos, nil
		}

		vars["prevEndCursor"] = data.Org.Repos.PageInfo.EndCursor
	}

	panic("should not get here")
}
