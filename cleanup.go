package worktree

import "fmt"

type Confirmer interface {
	Confirm(title, description string) (bool, error)
}

// Runner executes a single command.
type Runner func(Command) error

// MergedWorktrees returns the worktrees flagged as merged (see MarkMerged).
func MergedWorktrees(worktrees []Worktree) (results []Worktree) {
	for _, wt := range worktrees {
		if wt.Merged {
			results = append(results, wt)
		}
	}
	return results
}

// Repositories returns one directory per distinct repository among dirs,
// preserving order. gitDir reports the repository (the common git directory)
// that a directory belongs to; directories for which it fails are skipped.
func Repositories(dirs []string, gitDir func(dir string) (string, error)) (results []string) {
	seen := make(map[string]bool)
	for _, dir := range dirs {
		repo, err := gitDir(dir)
		if err != nil || seen[repo] {
			continue
		}
		seen[repo] = true
		results = append(results, dir)
	}
	return results
}

// CleanUp offers, one at a time, to delete each candidate worktree (and its
// branch), running the removal commands immediately upon confirmation. The
// commands run against mainPath (the main worktree) so that `git branch -d`
// checks for merging against the main branch rather than whatever happens to
// be checked out in the working directory. A failure to remove one worktree
// is collected in failures and doesn't prevent offering the rest; err is only
// returned if prompting fails.
func CleanUp(mainPath string, candidates []Worktree, confirmer Confirmer, run Runner) (failures []error, err error) {
	for i, wt := range candidates {
		title := fmt.Sprintf("Delete merged worktree %d of %d?", i+1, len(candidates))
		description := fmt.Sprintf("%s (%s)", wt.Branch, wt.Path)
		confirmed, err := confirmer.Confirm(title, description)
		if err != nil {
			return failures, err
		}
		if !confirmed {
			continue
		}
		for _, command := range cleanUpCommands(mainPath, wt) {
			if err := run(command); err != nil {
				failures = append(failures, fmt.Errorf("%s: %w", command, err))
				break
			}
		}
	}
	return failures, nil
}

func cleanUpCommands(mainPath string, wt Worktree) (results []Command) {
	return []Command{
		{Name: "git", Args: []string{"-C", mainPath, "worktree", "remove", wt.Path}},
		{Name: "git", Args: []string{"-C", mainPath, "branch", "-d", wt.Branch}},
	}
}
