package github_test

import (
	"testing"

	"github.com/mikesep/frond/internal/git"
	"github.com/mikesep/frond/internal/github"
)

func Test_GitHub(t *testing.T) {
	cred, err := git.FillCredential("https", "github.com")
	if err != nil {
		panic(err)
	}

	sat := github.ServerAndToken{
		Server: "github.com",
		Token:  cred.Password,
	}

	repos, err := sat.OrgRepos("golang")

	t.Logf("err: %+v", err)
	t.Logf("repos: %+v ...", repos[0:10])
	t.Logf("repos len: %d ", len(repos))
}
