// SPDX-FileCopyrightText: 2020 Michael Seplowitz
// SPDX-License-Identifier: MIT

package github_test

import (
	"bytes"
	"context"
	"testing"

	"github.com/mikesep/frond/internal/git"
	"github.com/mikesep/frond/internal/github"
	"github.com/stretchr/testify/assert"
)

func Test_ListRepos(t *testing.T) {
	cred, err := git.FillCredential("https", "github.com")
	if err != nil {
		panic(err)
	}

	sat := github.ServerAndToken{
		Server: "github.com",
		Token:  cred.Password,
	}

	ctx := context.Background()

	var progressBuffer bytes.Buffer

	repos, err := sat.ListRepos(ctx, &progressBuffer,
		github.Account{Login: "bloomberg", Type: "Organization"}, github.AllRepos)

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

func Test_GetAccount(t *testing.T) {
	cred, err := git.FillCredential("https", "github.com")
	if err != nil {
		panic(err)
	}

	sat := github.ServerAndToken{
		Server: "github.com",
		Token:  cred.Password,
	}

	ctx := context.Background()

	t.Run("bloomberg", func(t *testing.T) {
		acct, err := sat.GetAccount(ctx, "bloomberg")
		assert.Equal(t, "bloomberg", acct.Login)
		assert.Equal(t, "Organization", acct.Type)
		assert.NoError(t, err)
	})

	t.Run("mikesep", func(t *testing.T) {
		acct, err := sat.GetAccount(ctx, "mikesep")
		assert.Equal(t, "mikesep", acct.Login)
		assert.Equal(t, "User", acct.Type)
		assert.NoError(t, err)
	})

	t.Run("invalid", func(t *testing.T) {
		acct, err := sat.GetAccount(ctx, "name with spaces")
		assert.Zero(t, acct)
		assert.Error(t, err)
	})
}
