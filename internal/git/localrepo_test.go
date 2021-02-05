package git_test

import (
	"testing"

	"github.com/mikesep/frond/internal/git"
)

func Test_IsLocalRepoRoot(t *testing.T) {
	r, err := git.IsLocalRepoRoot("../../")
	t.Logf("r = %+v", r)
	t.Logf("err = %v", err)
}
