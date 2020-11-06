package github_test

import (
	"context"
	"testing"

	"github.com/mikesep/frond/internal/git"
	"github.com/mikesep/frond/internal/github"
)

func Test_ReposInOrg(t *testing.T) {
	cred, err := git.FillCredential("https", "github.com")
	if err != nil {
		panic(err)
	}

	sat := github.ServerAndToken{
		Server: "github.com",
		Token:  cred.Password,
	}

	ctx := context.Background()

	langGo := func(r github.Repo) bool {
		// t.Logf("langGo: %s r.Language=%v", r.Name, r.Language)
		return r.Language == "Go"
	}

	repos, err := sat.ReposInOrg(ctx, "bloomberg", github.AllRepos, langGo)

	t.Logf("err: %+v", err)
	t.Logf("repos len: %d ", len(repos))

	limit := 200
	if limit > len(repos) {
		limit = len(repos)
	}

	for i, r := range repos[0:limit] {
		t.Logf("  %2d: %s %s\n", i, r.FullName, r.Language)
	}
}
