# Proposal: Delete Worktree Command

## Background

The worktree tool currently supports two actions: selecting an existing worktree (emits `cd`) and creating a new one (emits `git branch` + `git worktree add` + `cd`). There's no way to clean up worktrees you're done with. Deleting a worktree requires remembering the `git worktree remove` command and the correct path — exactly the kind of thing this tool is meant to simplify.

## Approach

### UX Flow

Add a "Delete a worktree" option to the interactive menu, alongside the existing "Create new worktree" option. When selected:

1. Present a second selection menu showing only the non-main worktrees (you shouldn't delete the main worktree).
2. The user picks which worktree to delete.
3. Emit: `git worktree remove "<path>"; git branch -d <branch>`

This removes the worktree directory and deletes the associated branch. Using `git branch -d` (lowercase) ensures the branch is only deleted if it has been merged — a safety net against losing unmerged work. The user can always re-run with `-D` manually if they know what they're doing.

### Changes to the Menu

The top-level menu becomes:

- **"Create new worktree"** (index 0)
- One entry per existing worktree (indices 1 through N)
- **"Delete a worktree"** (last index, N+1)

This keeps the existing index math unchanged — worktree entries are still at `selected-1` offset from the worktrees slice. The delete option sits at the bottom of the list.

When no worktrees exist, the current behavior (skip straight to create) remains unchanged — there's nothing to delete.

### Changes to Code

- **`worktree.go`**: Update `Run` to append "Delete a worktree" to the options list, handle `selected == len(worktrees)+1` by calling a new `deleteWorktree` function. The delete function presents a sub-menu of worktrees and emits the removal commands.
- **`worktree_test.go`**: New tests for the delete flow and edge cases.
- **No changes to `main.go` or `prompter.go`** — the existing `Prompter` interface already supports everything needed.

### Emitted Command

```
git worktree remove "<path>"; git branch -d <branch>
```

### Files Modified

| File                | Change                                                |
|---------------------|-------------------------------------------------------|
| `worktree.go`       | Add delete option to menu, implement `deleteWorktree` |
| `worktree_test.go`  | Tests for delete flow                                 |

## Trade-offs & Risks

- **`-d` vs `-D`**: Using `-d` is safer (refuses to delete unmerged branches) but may surprise the user if the branch isn't merged yet. The emitted command is visible before execution (via `pbcopy` or inspection), so the user can modify it. This seems like the right default.
- **Deleting the main worktree**: The delete sub-menu should exclude the main worktree (the first entry from `git worktree list`, which is the bare repo root). If somehow only the main worktree exists, the "Delete a worktree" option shouldn't appear — but this is the same as the "no worktrees" shortcut already in place.
- **No index shift for existing tests**: Since "Delete a worktree" is appended at the end (after worktree entries), the existing index math for "Create new" (0) and worktree selection (1+) is unchanged. Existing tests should not need updating.

## Implementation Checklist

### Phase 1: Add delete option to menu

- [x] Update `Run` to append "Delete a worktree" to the options list after worktree entries
- [x] Run tests, confirm existing tests still pass (no index shift)

### Phase 2: Delete worktree flow

- [x] Write test: user selects "Delete a worktree" (last index), picks a worktree from sub-menu → expect `git worktree remove "<path>"; git branch -d <branch>` output — expect failure
- [x] Run tests, confirm failure
- [x] Implement `deleteWorktree` function and wire it in `Run`
- [x] Run tests, confirm passing

### Phase 3: Edge case — only main worktree

- [x] Write test: when only the main worktree exists (single entry), skip straight to create (same as no-worktrees behavior, since main is excluded from delete candidates) — expect failure
- [x] Run tests, confirm failure
- [x] Implement: filter out main worktree from delete candidates; if no candidates, treat as no-worktrees shortcut
- [x] Run tests, confirm passing

### Phase 4: Polish

- [x] Run `make test`, confirm passing
- [x] Review and audit code before committing
