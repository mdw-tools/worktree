# Proposal: Interactive Git Worktree Manager

## Background

The goal is to build an interactive terminal program (`worktree`) that simplifies git worktree management. Users currently must remember and type multi-step git commands to list, switch between, or create worktrees. This tool will present an interactive selection menu, then emit shell commands (for piping to `bash` or `pbcopy`) that perform the chosen action.

The repo already has a Go 1.25 module with a skeleton `cmd/worktree/main.go` and a `Makefile`.

## Approach

### Architecture

Separate the program into three layers:

1. **CLI (`cmd/worktree/main.go`)** — Parses flags, wires dependencies, runs the app, prints the resulting shell script to stdout.
2. **Core logic (`worktree.go` at module root)** — Pure function(s) that orchestrate the flow: gather worktrees, present choices, and produce the shell command string. All I/O is injected via interfaces/functions so the core is fully testable without git or a real terminal.
3. **Git integration** — A thin adapter that shells out to `git worktree list` and `git rev-parse` to gather repository state.

### Key Design Decisions

- **Interfaces for I/O**: The core accepts interfaces for git operations and terminal interaction (prompt/select). Tests supply fakes; `main.go` supplies real implementations.
- **No direct git library dependency**: Shell out to `git` — it's universally available, simple, and avoids a heavy dependency for a handful of commands.
- **Interactive selection via `charmbracelet/huh`**: Use `charmbracelet/huh` for the interactive selection menu and text input prompts. The core logic won't depend on it directly — it will be injected via interfaces.
- **Output is a shell script**: The program prints shell commands to stdout. It does not execute them itself. This matches the README's intent of piping to `bash` or `pbcopy`.

### CLI Flags

| Flag        | Default      | Description                           |
|-------------|--------------|---------------------------------------|
| `--user`    | `mikewhat`   | User branch prefix                    |
| `--workdir` | `$HOME/work` | Working path prefix for new worktrees |

### Flow

1. Detect the git repo root (`git rev-parse --show-toplevel`) and basename.
2. List existing worktrees (`git worktree list --porcelain`).
3. If no additional worktrees exist, skip directly to step 5a.
4. Present an interactive menu:
   - **"Create new worktree"** (topmost option)
   - One entry per existing worktree (showing branch name and path)
5. If the user selects an existing worktree → emit `cd "<path>"`.
6. If the user selects "Create new worktree":
   a. Prompt for a feature name (validate: alphanumeric + hyphens only).
   b. Emit: `git branch <user>/<feature>; git worktree add <workdir>/<project-basename>/<feature> <user>/<feature>; cd "$_"`

### File Plan

| File                   | Purpose                              |
|------------------------|--------------------------------------|
| `cmd/worktree/main.go` | CLI entry point — flags, wiring, run |
| `worktree.go`          | Core orchestration logic             |
| `worktree_test.go`     | Tests for core logic                 |
| `git.go`               | Git adapter (shell out to git)       |

### Alternatives Considered

- **Use `go-git` library**: Rejected — adds a large dependency for trivial operations. Shelling out to `git` is simpler and more reliable for worktree commands.
- **Execute commands directly instead of emitting script**: Rejected — the README explicitly calls for emitting commands to stdout for piping. Also, `cd` cannot affect the parent shell from a child process, so emitting is the correct approach.
- **Single-file implementation**: Rejected — separating core logic from I/O enables thorough testing without git repos or terminal interaction.

## Trade-offs & Risks

- **TUI library choice**: We'll need to pick a library for interactive selection. If the chosen library doesn't fit well, swapping it out is straightforward since the core logic is decoupled.
- **Git command parsing**: Parsing `git worktree list --porcelain` output is straightforward but brittle if git changes its format. The porcelain format is designed to be stable, so this is low risk.
- **Feature name validation**: The README specifies "only alpha-numerics and hyphens, no spaces". We'll enforce this at prompt time and reject invalid input.
- **No existing worktrees edge case**: If the repo has no additional worktrees, skip the menu entirely and go straight to the "create new worktree" prompt.

## Implementation Checklist

### Phase 1: Core types and git adapter

- [x] Define the core interfaces/types in `worktree.go` (e.g., `Worktree` struct, git interface, prompter interface)
- [x] Write test for parsing `git worktree list --porcelain` output — expect failure (no implementation)
- [x] Run tests, confirm failure
- [x] Implement the porcelain output parser
- [x] Run tests, confirm passing

### Phase 2: Existing worktree selection

- [x] Write test: given a list of worktrees and user selecting one, expect `cd "<path>"` output — expect failure
- [x] Run tests, confirm failure
- [x] Implement core orchestration logic for the selection-and-emit flow
- [x] Run tests, confirm passing

### Phase 3: New worktree creation

- [x] Write test: user selects "Create new worktree", provides feature name → expect correct `git branch ...; git worktree add ...; cd "$_"` output — expect failure
- [x] Run tests, confirm failure
- [x] Implement the new-worktree branch of the orchestration logic
- [x] Run tests, confirm passing

### Phase 4: Feature name validation

- [x] Write test: invalid feature names (spaces, special chars) are rejected — expect failure
- [x] Run tests, confirm failure
- [x] Implement validation logic
- [x] Run tests, confirm passing

### Phase 5: No-worktrees shortcut

- [x] Write test: when no additional worktrees exist, skip menu and go straight to feature name prompt — expect failure
- [x] Run tests, confirm failure
- [x] Implement the shortcut path
- [x] Run tests, confirm passing

### Phase 6: CLI wiring

- [x] Wire flags (`--user`, `--workdir`) in `main.go`
- [x] Wire real git adapter and real TUI prompter
- [x] Implement git adapter (shell out to `git worktree list --porcelain`, `git rev-parse --show-toplevel`)
- [x] Manual smoke test: run the tool in a git repo, verify interactive menu and output

### Phase 7: Polish

- [x] Ensure `make test` passes cleanly
- [x] Review and audit code before committing
