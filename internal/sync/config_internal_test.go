// SPDX-FileCopyrightText: 2021 Michael Seplowitz
// SPDX-License-Identifier: MIT

package sync

import (
	"strings"
	"testing"

	"github.com/bloomberg/go-testgroup"
)

var (
	falseVar = false
	trueVar  = true
)

func Test_syncconfig(t *testing.T) {
	testgroup.RunInParallel(t, &syncConfigTests{})
}

type syncConfigTests struct{}

func (grp *syncConfigTests) Decode_single_org(t *testgroup.T) {
	cfg, err := parseConfig(strings.NewReader(`
github:
  server: github.com
  org: bloomberg
`))

	t.Require.NoError(err)

	t.Require.NotNil(cfg.GitHub)
	gh := cfg.GitHub

	t.Equal("github.com", gh.Server)
	t.Equal("bloomberg", gh.SingleOrg)
}

func (grp *syncConfigTests) Decode_orgs_with_global_criteria(t *testgroup.T) {
	cfg, err := parseConfig(strings.NewReader(`
github:
  server: github.com
  orgs:
    apache:
    bloomberg:
    containers:
  languages: [go]
  archived: false
  fork: false
  exclude:
    names: [airflow, bookkeeper, couchdb]
`))
	t.Require.NoError(err)

	t.Require.NotNil(cfg.GitHub)
	gh := cfg.GitHub

	t.Equal("github.com", gh.Server)

	t.Contains(gh.Orgs, "apache")
	t.Contains(gh.Orgs, "bloomberg")
	t.Contains(gh.Orgs, "containers")

	t.Empty(gh.Names)

	t.Empty(gh.Topics)
	t.Equal([]string{"go"}, gh.Languages)

	t.Equal(&falseVar, gh.Archived)
	t.Equal(&falseVar, gh.Fork)
	t.Nil(gh.IsTemplate)
	t.Nil(gh.Private)

	t.Nil(gh.SingleDirForAllRepos)
	t.Nil(gh.AccountPrefixSeparator)

	t.Require.NotNil(gh.Exclude)
	t.Contains(gh.Exclude.Names, "airflow")
	t.Contains(gh.Exclude.Names, "bookkeeper")
	t.Contains(gh.Exclude.Names, "couchdb")
}

func (grp *syncConfigTests) Decode_org_with_criteria(t *testgroup.T) {
	cfg, err := parseConfig(strings.NewReader(`
github:
  server: github.com
  orgs:
    apache:
      languages: [java]
      archived: false
      exclude:
        names: [zookeeper]
`))

	t.Require.NoError(err)

	t.Require.NotNil(cfg.GitHub)
	gh := cfg.GitHub

	t.Equal("github.com", gh.Server)

	t.Len(gh.Orgs, 1)
	t.Require.Contains(gh.Orgs, "apache")
	apache := gh.Orgs["apache"]
	t.Require.NotNil(apache)

	t.Equal(&falseVar, apache.Archived)
	t.Equal([]string{"java"}, apache.Languages)

	t.Require.NotNil(apache.Exclude)
	t.Equal([]string{"zookeeper"}, apache.Exclude.Names)
}
