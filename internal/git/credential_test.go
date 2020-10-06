package git_test

import (
	"testing"

	"github.com/bloomberg/go-testgroup"
	"github.com/mikesep/frond/internal/git"
)

func Test_Credential(t *testing.T) {
	testgroup.RunInParallel(t, &CredentialTests{})
}

type CredentialTests struct{}

func (*CredentialTests) Fill_for_github(t *testgroup.T) {
	cred, err := git.FillCredential("https", "github.com")

	t.NoError(err)
	t.Equal("https", cred.Protocol)
	t.Equal("github.com", cred.Host)
	t.NotEmpty(cred.Username)
	t.True(cred.Password != "", "cred.Password was empty") // use True to avoid exposing password
}
