# worktree

An interactive terminal program for managing git worktrees.

## Operation

At startup, the tool detects any existing git worktrees for the repository associated with the working directory.

All worktrees are displayed for selection, as well as an option to create a new worktree (that option should probably be the topmost in the list).

If the user selects one, emit a `cd` command to its location on disk and exit.

If the user indicates they want to create a new worktree, ask the user to supply the hyphenated feature-name (only alpha-numerics and hyphens, no spaces allowed) to use with the branch.

Then, emit the following script and exit:

`git branch <user-prefix>/<feature-name>; git worktree add <working-path-prefix>/<project-basename>/<feature-name>; cd "_$"`

- The <user-prefix> defaults to `mikewhat`, but can be modified by a CLI flag.
- The <feature-name> is provided by the user (as previously specified)
- The <working-path-prefix> defaults to `$HOME/work`, but can be overriden via CLI flag.
- The project basename comes from the basename of the root directory of the current git repo.


It is intended that the user will pipe the output of this program to `pbcopy` for inspection or `bash` for immediate execution.