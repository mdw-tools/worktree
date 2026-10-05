# worktree

An interactive terminal program for managing git worktrees.

## Operation

At startup, the tool detects any existing git worktrees and branches for the
repository associated with the working directory, then offers four options:

- **Enter worktree**: lists all worktrees for selection. The tool spawns a
  fresh shell (`$SHELL`) with its working directory set to the selected
  worktree. Type `exit` to return to the original shell.
- **Create new worktree**: asks for a hyphenated feature-name (only
  alpha-numerics and hyphens, no spaces allowed), then prints the git commands
  it intends to run and asks for confirmation:

  `git branch <user-prefix>/<feature-name>` followed by
  `git worktree add <working-path-prefix>/<project-basename>/<feature-name> <user-prefix>/<feature-name>`
- **Create worktree from branch**: lists every existing branch not already
  checked out in a worktree (local branches, then remote branches with no
  local counterpart) and, after confirmation, runs
  `git worktree add <working-path-prefix>/<project-basename>/<dir> <branch>`.
  For a remote branch it instead runs
  `git worktree add --track -b <branch> <working-path-prefix>/<project-basename>/<dir> <remote>/<branch>`.
  The `<dir>` is the branch name minus a leading `<user-prefix>/`, with any
  remaining slashes replaced by hyphens (e.g. `alice/fix` becomes `alice-fix`).
- **Delete worktree**: lists all worktrees except the main one, and runs
  `git worktree remove <path>` and `git branch -d <branch>` after confirmation.

On confirmation, commands are executed directly (via `exec.Command`), and when
a worktree was created the tool then spawns a shell in it.

- The <user-prefix> defaults to `mikewhat`, but can be modified by a CLI flag.
- The <feature-name> is provided by the user (as previously specified)
- The <working-path-prefix> defaults to `$CODEPATH/work`, but can be overriden via CLI flag.
- The project basename comes from the basename of the main worktree of the current git repo.

Worktrees whose branch has already been merged into the main worktree's branch
(per `git branch --merged`) are labeled `[merged]` in both the enter and delete
menus, as a hint about what can be deleted.

Because the git operations now run in-process behind a confirmation prompt, the
tool no longer needs to be piped to `bash` or `pbcopy`.

## worktree-cleanup

A separate tool for periodically sweeping away finished worktrees across all
projects. It can be run from anywhere:

1. It finds every repository with a worktree under `<workdir>/<project>/`
   (`-workdir` defaults to `$CODEPATH/work`, same as `worktree`).
2. For each repository, it lists all worktrees, labeling those whose branch
   has been merged into the main worktree's branch (per `git branch --merged`).
3. For each merged worktree, one at a time, it asks whether to delete it. On
   confirmation it immediately runs `git worktree remove <path>` and
   `git branch -d <branch>` (both against the main worktree).

A worktree that fails to delete (e.g. one with uncommitted changes, which
`git worktree remove` refuses) is reported at the end without stopping the
rest. Note that a branch with no commits of its own (e.g. one just created)
counts as merged, and squash-merged branches do not.

## Claude Code skill

This repository is also a Claude Code plugin (and a single-plugin marketplace)
providing a `worktree` skill, which teaches Claude to create, enter, and
delete worktrees using the same conventions as the tool. Install it with:

```
/plugin marketplace add mdw-tools/worktree
/plugin install worktree@worktree
```
