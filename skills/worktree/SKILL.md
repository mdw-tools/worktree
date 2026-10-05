---
name: worktree
description: Create, enter, list, and delete git worktrees using the conventions of the `worktree` tool (branch `<user>/<feature>`, directory `$CODEPATH/work/<project>/<feature>`). Use when the user asks to start work in a new or separate worktree, to check out an existing branch (for example a PR branch to review) without disturbing the current checkout, to find or switch to a worktree, or to clean up finished worktrees.
---

The user manages git worktrees with `worktree`, an interactive terminal tool. Claude cannot drive its menus, so run the equivalent git commands directly, following the same conventions so that the user's tool recognizes and lays out everything the same way.

## Conventions

| Value          | Rule                                                                                       |
|----------------|--------------------------------------------------------------------------------------------|
| `<user>`       | Branch prefix. Defaults to `mikewhat` (the tool's `-user` flag).                           |
| `<workdir>`    | Parent of all worktrees. Defaults to `$CODEPATH/work` (the tool's `-workdir` flag).        |
| `<main>`       | Path of the main worktree: the first `worktree` line of `git worktree list --porcelain`.   |
| `<project>`    | Basename of `<main>`. Do not use `git rev-parse --show-toplevel` from a linked worktree.  |
| `<feature>`    | Alphanumerics and hyphens only (`^[a-zA-Z0-9]+(-[a-zA-Z0-9]+)*$`).                         |

If `$CODEPATH` is unset, ask the user where worktrees belong rather than guessing.

## When to use a worktree

- Use one when the user asks for it.
- Suggest one (and wait for a yes) when the user wants to start unrelated work while the current tree has uncommitted changes or work in progress, or wants to review or test another branch without disturbing the current checkout.
- Do not create worktrees unprompted for routine work in the current checkout.

## List worktrees

```
git worktree list --porcelain
git -C <main> branch --merged <main-branch> --format='%(refname:short)'
```

A non-main worktree whose branch appears in the second listing is merged and likely safe to delete. A branch with no commits of its own also appears there, and squash-merged branches do not.

## Create a new worktree

Validate `<feature>` against the pattern above, then run:

```
git branch <user>/<feature>
git worktree add <workdir>/<project>/<feature> <user>/<feature>
```

`git branch` starts the new branch at the current `HEAD`, just like the tool. If the user names a different base, pass it as a start point: `git branch <user>/<feature> <base>`.

## Create a worktree from an existing branch

Skip any branch that is already checked out in a worktree (git refuses to check it out twice). The directory name is the branch name with a leading `<user>/` removed and any remaining `/` replaced by `-`. For example, `mikewhat/fix-login` becomes `fix-login` and `alice/fix/flaky-test` becomes `alice-fix-flaky-test`.

For a local branch:

```
git worktree add <workdir>/<project>/<dir> <branch>
```

For a branch that exists only on a remote (run `git fetch <remote>` first if the user wants the latest), create a local branch that tracks it:

```
git worktree add --track -b <branch> <workdir>/<project>/<dir> <remote>/<branch>
```

## Enter a worktree

Claude cannot hand the user a new shell. To work in a worktree, `cd` into its path (the Bash tool's working directory persists) or use absolute paths under it. Tell the user the path so they can follow with `cd <path>` or with `worktree` and its "Enter worktree" option.

## Delete a worktree

Deleting is destructive, so always confirm with the user first and name the worktree path and branch. Never delete the main worktree. Run the commands against the main worktree so that `git branch -d` checks merging against the main branch:

```
git -C <main> worktree remove <path>
git -C <main> branch -d <branch>
```

Never add `--force` to `git worktree remove` or use `git branch -D` unless the user explicitly asks. If git refuses (for uncommitted changes or an unmerged branch), report why and let the user decide. If Claude's working directory is inside the worktree being removed, `cd <main>` first.

For a sweep of every merged worktree across all projects, suggest the user run `worktree-cleanup`.
