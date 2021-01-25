# frond :palm_tree:

## multi-repo actions

```console
$ frond status
$ frond status orgname
$ frond status path/to/repo orgnames*

$ frond ls # all repos
$ frond ls --changed
$ frond ls --stashed
$ frond ls --untracked
$ frond ls --unmerged
$ frond ls --ignored
$ frond ls --changed --unmerged # OR'd together

$ frond ls --branch=<glob/regexp>

$ frond pr

$ frond do git pull --prune
  $ # is the equivalent of
  $ frond ls | parallel cd {} "&&" git pull --prune
  $ # or serially
  $ for dir in $(frond ls) ; do (cd "$dir" && git pull --prune); done
```

## Sync

### Single org/user

```console
$ frond sync init github.com/bloomberg
Created frond.sync.yaml.
```

### Multi-org/user

```console
$ frond sync init github.com/apache github.com/bloomberg github.com/containers
$ frond sync init github.com/{apache,bloomberg,containers} # bash-ism

$ frond sync init github.com/apache,bloomberg,containers
$ frond sync init github.com apache bloomberg containers
$ frond sync init github.com --orgs=apache,bloomberg,containers
$ frond sync init github.com apache,bloomberg,containers

$ frond sync init gitlab.com/bloomberg
$ frond sync init gitlab.com/bloomberg/bst
```

---

```console
$ frond sync init github.com "org:bloomberg language:go"
Created frond.sync.yaml.

$ frond sync
Downloading...

$ frond sync orgA orgB orgC/specific-repo

$ frond sync prune [--dry-run]
Would remove:
org/archived-repo
org/repo-to-avoid

$ frond sync prune --force
Removing:
org/archived-repo
org/repo-to-avoid

$ frond sync [--prune-repos] # clone, pull --prune, and warn/delete
```

Tom's scenario

1. work on branch foo
2. push foo and create PR
3. later merge PR and delete branch
4. working repo gets left on branch
5. sync up by getting rid of the local and remote branch and switching back to
   default branch
