// SPDX-FileCopyrightText: 2021 Michael Seplowitz
// SPDX-License-Identifier: MIT

package sync

import (
	"path/filepath"
	"strconv"
	"testing"

	"github.com/bloomberg/go-testgroup"
)

func Test_GitHub_paths(t *testing.T) {
	testgroup.RunInParallel(t, &gitHubPathTests{})
}

type gitHubPathTests struct{}

type gitHubPathTestcase struct {
	path string

	org  string
	user string
	repo string
}

//------------------------------------------------------------------------------

func (grp *gitHubPathTests) ManyOrgsAndUsers(t *testgroup.T) {
	t.RunInParallel(&manyOrgsAndUsersTests{
		cfg: gitHubConfig{
			Orgs: map[string]*gitHubConfigCriteriaWithExclusions{
				"apache":     nil,
				"bloomberg":  nil,
				"containers": nil,
			},
			Users: map[string]*gitHubConfigCriteriaWithExclusions{
				"alice":   nil,
				"bob":     nil,
				"charlie": nil,
			},
		},
	})
}

type manyOrgsAndUsersTests struct {
	cfg gitHubConfig
}

func (grp *manyOrgsAndUsersTests) Panics(t *testgroup.T) {
	t.Panics(func() {
		grp.cfg.pathToOrgUserRepo("")
	})
}

func (grp *manyOrgsAndUsersTests) Unmatched(t *testgroup.T) {
	cfg := grp.cfg

	for i, path := range []string{"unknown", "unknown/repo"} {
		t.Run(strconv.Itoa(i), func(t *testgroup.T) {
			org, user, repo := cfg.pathToOrgUserRepo(path)
			t.Empty(org)
			t.Empty(user)
			t.Empty(repo)
		})
	}

	t.Panics(func() {
		cfg.pathForRepo("unknown", "repo")
	})
}

func (grp *manyOrgsAndUsersTests) Simple(t *testgroup.T) {
	cfg := grp.cfg

	for i, c := range []gitHubPathTestcase{
		{
			path: "apache",
			org:  "apache",
		},
		{
			path: filepath.Join("apache", "repo"),
			org:  "apache",
			repo: "repo",
		},
		{
			path: "alice",
			user: "alice",
		},
		{
			path: filepath.Join("alice", "repo"),
			user: "alice",
			repo: "repo",
		},
	} {
		t.Run(strconv.Itoa(i), func(t *testgroup.T) {
			org, user, repo := cfg.pathToOrgUserRepo(c.path)
			t.Equal(c.org, org)
			t.Equal(c.user, user)
			t.Equal(c.repo, repo)

			acct := c.org + c.user
			t.Equal(c.path, cfg.pathForRepo(acct, c.repo))
		})
	}

}

func (grp *manyOrgsAndUsersTests) SingleDir(t *testgroup.T) {
	cfg := grp.cfg

	trueVar := true
	cfg.SingleDirForAllRepos = &trueVar

	for i, c := range []gitHubPathTestcase{
		{
			path: "containers__repo",
			org:  "containers",
			repo: "repo",
		},
		{
			path: "alice__repo",
			user: "alice",
			repo: "repo",
		},
	} {
		t.Run(strconv.Itoa(i), func(t *testgroup.T) {
			org, user, repo := cfg.pathToOrgUserRepo(c.path)
			t.Equal(c.org, org)
			t.Equal(c.user, user)
			t.Equal(c.repo, repo)

			acct := c.org + c.user
			t.Equal(c.path, cfg.pathForRepo(acct, c.repo))
		})
	}
}

func (grp *manyOrgsAndUsersTests) SingleDirArrow(t *testgroup.T) {
	cfg := grp.cfg

	trueVar := true
	cfg.SingleDirForAllRepos = &trueVar

	arrow := "-->"
	cfg.AccountPrefixSeparator = &arrow

	for i, c := range []gitHubPathTestcase{
		{
			path: "containers" + arrow + "repo",
			org:  "containers",
			repo: "repo",
		},
		{
			path: "alice" + arrow + "repo",
			user: "alice",
			repo: "repo",
		},
	} {
		t.Run(strconv.Itoa(i), func(t *testgroup.T) {
			org, user, repo := cfg.pathToOrgUserRepo(c.path)
			t.Equal(c.org, org)
			t.Equal(c.user, user)
			t.Equal(c.repo, repo)

			acct := c.org + c.user
			t.Equal(c.path, cfg.pathForRepo(acct, c.repo))
		})
	}
}

func (grp *manyOrgsAndUsersTests) ManyDirsWithArrow(t *testgroup.T) {
	cfg := grp.cfg

	arrow := "-->"
	cfg.AccountPrefixSeparator = &arrow

	for i, c := range []gitHubPathTestcase{
		{
			path: filepath.Join("containers", "containers"+arrow+"repo"),
			org:  "containers",
			repo: "repo",
		},
		{
			path: filepath.Join("alice", "alice"+arrow+"repo"),
			user: "alice",
			repo: "repo",
		},
	} {
		t.Run(strconv.Itoa(i), func(t *testgroup.T) {
			org, user, repo := cfg.pathToOrgUserRepo(c.path)
			t.Equal(c.org, org)
			t.Equal(c.user, user)
			t.Equal(c.repo, repo)

			acct := c.org + c.user
			t.Equal(c.path, cfg.pathForRepo(acct, c.repo))
		})
	}
}

//------------------------------------------------------------------------------

func (grp *gitHubPathTests) SingleOrg(t *testgroup.T) {
	t.RunInParallel(&singleAccountTests{
		cfg: gitHubConfig{
			SingleOrg: "golang",
		},
	})
}

func (grp *gitHubPathTests) SingleUser(t *testgroup.T) {
	t.RunInParallel(&singleAccountTests{
		cfg: gitHubConfig{
			SingleUser: "alice",
		},
	})
}

type singleAccountTests struct {
	cfg gitHubConfig
}

func (grp *singleAccountTests) Simple(t *testgroup.T) {
	cfg := grp.cfg

	org, user, repo := cfg.pathToOrgUserRepo("repo")

	t.Equal(cfg.SingleOrg, org)
	t.Equal(cfg.SingleUser, user)
	t.Equal("repo", repo)

	acct := cfg.SingleOrg + cfg.SingleUser
	t.Equal("repo", cfg.pathForRepo(acct, "repo"))
}

func (grp *singleAccountTests) CustomSeparator(t *testgroup.T) {
	cfg := grp.cfg

	arrow := "-->"
	cfg.AccountPrefixSeparator = &arrow

	acct := cfg.SingleOrg + cfg.SingleUser
	path := acct + arrow + "repo"

	t.Equal(path, cfg.pathForRepo(acct, "repo"))

	org, user, repo := cfg.pathToOrgUserRepo(path)
	t.Equal(cfg.SingleOrg, org)
	t.Equal(cfg.SingleUser, user)
	t.Equal("repo", repo)

	// prefix doesn't match single account
	org, user, repo = cfg.pathToOrgUserRepo("different" + arrow + "repo")
	t.Empty(org)
	t.Empty(user)
	t.Empty(repo)
}

func (grp *singleAccountTests) InNestedDir(t *testgroup.T) {
	cfg := grp.cfg

	falseVar := false
	cfg.SingleDirForAllRepos = &falseVar

	acct := cfg.SingleOrg + cfg.SingleUser
	path := filepath.Join(acct, "repo")

	org, user, repo := cfg.pathToOrgUserRepo(path)
	t.Equal(cfg.SingleOrg, org)
	t.Equal(cfg.SingleUser, user)
	t.Equal("repo", repo)

	t.Equal(path, cfg.pathForRepo(acct, "repo"))
}

func (grp *singleAccountTests) InNestedDirWithSeparator(t *testgroup.T) {
	cfg := grp.cfg

	falseVar := false
	cfg.SingleDirForAllRepos = &falseVar

	arrow := "-->"
	cfg.AccountPrefixSeparator = &arrow

	acct := cfg.SingleOrg + cfg.SingleUser
	path := filepath.Join(acct, acct+arrow+"repo")

	org, user, repo := cfg.pathToOrgUserRepo(path)
	t.Equal(cfg.SingleOrg, org)
	t.Equal(cfg.SingleUser, user)
	t.Equal("repo", repo)

	t.Equal(path, cfg.pathForRepo(acct, "repo"))
}

func (grp *singleAccountTests) PanicsWhenDirAndPrefixConflict(t *testgroup.T) {
	cfg := grp.cfg

	falseVar := false
	cfg.SingleDirForAllRepos = &falseVar

	arrow := "-->"
	cfg.AccountPrefixSeparator = &arrow

	t.Panics(func() {
		acct := cfg.SingleOrg + cfg.SingleUser
		cfg.pathToOrgUserRepo(filepath.Join(acct, "different"+arrow+"repo"))
	})
}
