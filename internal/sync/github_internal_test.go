package sync

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_pathToOrgUserRepo(t *testing.T) {
	type testcase struct {
		path         string
		expectedOrg  string
		expectedUser string
		expectedRepo string
	}

	multiCfg := gitHubConfig{
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
	}

	for i, c := range []testcase{
		{
			path:        "apache",
			expectedOrg: "apache",
		},
		{
			path:         "apache/repo",
			expectedRepo: "apache/repo",
		},
		{
			path:         "alice",
			expectedUser: "alice",
		},
		{
			path:         "alice/repo",
			expectedRepo: "alice/repo",
		},
	} {
		t.Run(fmt.Sprintf("multiCfg_1_%d", i), func(t *testing.T) {
			org, user, repo := multiCfg.pathToOrgUserRepo(c.path)
			assert.Equal(t, c.expectedOrg, org)
			assert.Equal(t, c.expectedUser, user)
			assert.Equal(t, c.expectedRepo, repo)
		})
	}

	trueVar := true
	multiCfg.SingleDirForAllRepos = &trueVar

	for i, c := range []testcase{
		{
			path:         "containers__repo",
			expectedRepo: "containers/repo",
		},
		{
			path:         "alice__repo",
			expectedRepo: "alice/repo",
		},
	} {
		t.Run(fmt.Sprintf("multiDirCfg_singleDir_%d", i), func(t *testing.T) {
			org, user, repo := multiCfg.pathToOrgUserRepo(c.path)
			assert.Equal(t, c.expectedOrg, org)
			assert.Equal(t, c.expectedUser, user)
			assert.Equal(t, c.expectedRepo, repo)
		})
	}

	customSeparator := "-->"
	multiCfg.AccountPrefixSeparator = &customSeparator

	for i, c := range []testcase{
		{
			path:         fmt.Sprintf("containers%srepo", customSeparator),
			expectedRepo: "containers/repo",
		},
		{
			path:         fmt.Sprintf("alice%srepo", customSeparator),
			expectedRepo: "alice/repo",
		},
	} {
		t.Run(fmt.Sprintf("multiDirCfg_singleDir_arrow_%d", i), func(t *testing.T) {
			org, user, repo := multiCfg.pathToOrgUserRepo(c.path)
			assert.Equal(t, c.expectedOrg, org)
			assert.Equal(t, c.expectedUser, user)
			assert.Equal(t, c.expectedRepo, repo)
		})
	}

	singleOrgCfg := gitHubConfig{
		SingleOrg: "golang",
	}

	for i, c := range []testcase{
		{
			path:         "repo",
			expectedRepo: "golang/repo",
		},
	} {
		t.Run(fmt.Sprintf("singleOrgCfg_%d", i), func(t *testing.T) {
			org, user, repo := singleOrgCfg.pathToOrgUserRepo(c.path)
			assert.Equal(t, c.expectedOrg, org)
			assert.Equal(t, c.expectedUser, user)
			assert.Equal(t, c.expectedRepo, repo)
		})
	}

	singleOrgCfg.AccountPrefixSeparator = &customSeparator
	for i, c := range []testcase{
		{
			path:         fmt.Sprintf("golang%srepo", customSeparator),
			expectedRepo: "golang/repo",
		},
	} {
		t.Run(fmt.Sprintf("singleOrgCfg_arrow_%d", i), func(t *testing.T) {
			org, user, repo := singleOrgCfg.pathToOrgUserRepo(c.path)
			assert.Equal(t, c.expectedOrg, org)
			assert.Equal(t, c.expectedUser, user)
			assert.Equal(t, c.expectedRepo, repo)
		})
	}
}
