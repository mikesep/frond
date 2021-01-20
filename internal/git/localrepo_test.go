package git_test

import (
	"testing"

	"github.com/mikesep/frond/internal/git"
)

func Test_LocalRepoAtDir(t *testing.T) {
	r, err := git.LocalRepoAtDir("../../")
	t.Logf("r = %+v", r)
	t.Logf("err = %v", err)
}
