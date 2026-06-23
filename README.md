# worktree

An interactive terminal program for managing git worktrees.

## Operation

At startup, the tool detects any existing git worktrees for the repository associated with the working directory.

All worktrees are displayed for selection, as well as an option to create a new worktree (that option should probably be the topmost in the list).

If the user selects an existing worktree, the tool spawns a fresh shell (`$SHELL`)
with its working directory set to that worktree. Type `exit` to return to the
original shell.

If the user indicates they want to create a new worktree, the tool asks for a
hyphenated feature-name (only alpha-numerics and hyphens, no spaces allowed),
then prints the git commands it intends to run and asks for confirmation:

`git branch <user-prefix>/<feature-name>` followed by
`git worktree add <working-path-prefix>/<project-basename>/<feature-name> <user-prefix>/<feature-name>`

On confirmation, those commands are executed directly (via `exec.Command`), and
the tool then spawns a shell in the new worktree.

- The <user-prefix> defaults to `mikewhat`, but can be modified by a CLI flag.
- The <feature-name> is provided by the user (as previously specified)
- The <working-path-prefix> defaults to `$HOME/work`, but can be overriden via CLI flag.
- The project basename comes from the basename of the root directory of the current git repo.

A worktree can also be deleted from the menu, which runs
`git worktree remove <path>` and `git branch -d <branch>` after confirmation.

Because the git operations now run in-process behind a confirmation prompt, the
tool no longer needs to be piped to `bash` or `pbcopy`.