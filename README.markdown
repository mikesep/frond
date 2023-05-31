<!--
SPDX-FileCopyrightText: 2020 Michael Seplowitz
SPDX-License-Identifier: MIT
-->

# frond :palm_tree:

## multi-repo actions

### List repo paths

```console
$ frond ls # all repos
$ frond ls --changed
$ frond ls --stashed
$ frond ls --untracked
$ frond ls --unmerged
$ frond ls --ignored
$ frond ls --changed --unmerged # OR'd together
$ frond ls --branch=<glob/regexp>
```

### Check repos' status

```console
$ frond status
$ frond status orgname
$ frond status path/to/repo orgnames*
```

### Do something in each repo

```console
$ frond do git pull --prune
  $ # is the equivalent of
  $ for dir in $(frond ls) ; do (cd "$dir" && git pull --prune); done
  $ frond ls | parallel cd {} "&&" git pull --prune
```

### Make PRs?

TODO Is this something that belongs here? Or is this for another tool.

## Sync

```console
$ frond sync init https://github.com/bloomberg
$ frond sync init https://github.com/apache https://github.com/bloomberg https://github.com/containers
$ frond sync init https://github.com/{apache,bloomberg,containers}  # bash-ism
```

### Tom's scenario

TODO demonstrate this

1. work on branch foo
2. push foo and create PR
3. later merge PR and delete branch
4. working repo gets left on branch
5. sync up by getting rid of the local and remote branch and switching back to
   default branch
