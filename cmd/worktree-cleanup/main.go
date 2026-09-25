package main

import (
	"errors"
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/huh"
	"github.com/mdw-tools/worktree"
)

var Version = "dev"

func main() {
	flags := flag.NewFlagSet(fmt.Sprintf("%s @ %s", filepath.Base(os.Args[0]), Version), flag.ExitOnError)
	workDir := flags.String("workdir", filepath.Join(os.Getenv("CODEPATH"), "work"), "Working path prefix under which worktrees live (<workdir>/<project>/<feature>)")
	flags.Usage = func() {
		_, _ = fmt.Fprintf(flags.Output(), "Usage of %s:\n", flags.Name())
		_, _ = fmt.Fprintf(flags.Output(), "%s [flags]\n", filepath.Base(os.Args[0]))
		_, _ = fmt.Fprintln(flags.Output(), "For each repository with worktrees under -workdir, offers (one at a time) to delete each worktree (and branch) merged into the main worktree's branch.")
		flags.PrintDefaults()
	}
	_ = flags.Parse(os.Args[1:])

	dirs, err := filepath.Glob(filepath.Join(*workDir, "*", "*"))
	if err != nil {
		log.Fatal(err)
	}
	repos := worktree.Repositories(dirs, gitCommonDir)
	if len(repos) == 0 {
		_, _ = fmt.Fprintf(os.Stderr, "No git worktrees found under %s.\n", *workDir)
		return
	}

	confirmer := &huhConfirmer{}
	var failures []error
	for _, dir := range repos {
		repoFailures, err := cleanUpRepository(dir, confirmer)
		failures = append(failures, repoFailures...)
		if errors.Is(err, huh.ErrUserAborted) {
			_, _ = fmt.Fprintln(os.Stderr, "aborted")
			break
		}
		if err != nil {
			failures = append(failures, err)
		}
	}

	if len(failures) > 0 {
		_, _ = fmt.Fprintln(os.Stderr)
		for _, failure := range failures {
			_, _ = fmt.Fprintln(os.Stderr, "failed:", failure)
		}
		os.Exit(1)
	}
}

// cleanUpRepository lists the worktrees of the repository containing dir and
// offers to delete each one that has been merged.
func cleanUpRepository(dir string, confirmer worktree.Confirmer) (failures []error, err error) {
	porcelain, err := gitOutput("-C", dir, "worktree", "list", "--porcelain")
	if err != nil {
		return nil, err
	}
	worktrees := worktree.ParsePorcelain(porcelain)
	if len(worktrees) == 0 || worktrees[0].Branch == "" {
		return nil, fmt.Errorf("%s: main worktree is not on a branch; cannot determine what has been merged", dir)
	}

	primary := worktrees[0]
	merged, err := gitOutput("-C", primary.Path, "branch", "--merged", primary.Branch, "--format=%(refname:short)")
	if err != nil {
		return nil, err
	}
	worktrees = worktree.MarkMerged(worktrees, merged)

	_, _ = fmt.Fprintf(os.Stderr, "\n== %s\n", filepath.Base(primary.Path))
	for _, wt := range worktrees {
		_, _ = fmt.Fprintln(os.Stderr, wt)
	}

	candidates := worktree.MergedWorktrees(worktrees)
	if len(candidates) == 0 {
		_, _ = fmt.Fprintf(os.Stderr, "No worktrees merged into %s.\n", primary.Branch)
		return nil, nil
	}
	return worktree.CleanUp(primary.Path, candidates, confirmer, runCommand)
}

type huhConfirmer struct{}

func (this *huhConfirmer) Confirm(title, description string) (result bool, err error) {
	err = huh.NewConfirm().
		Title(title).
		Description(description).
		Value(&result).
		Run()
	return result, err
}

func runCommand(command worktree.Command) error {
	_, _ = fmt.Fprintln(os.Stderr, "→", command)
	cmd := exec.Command(command.Name, command.Args...)
	cmd.Stdout = os.Stderr
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func gitCommonDir(dir string) (string, error) {
	return gitOutput("-C", dir, "rev-parse", "--path-format=absolute", "--git-common-dir")
}

func gitOutput(args ...string) (string, error) {
	out, err := exec.Command("git", args...).Output()
	if err != nil {
		return "", fmt.Errorf("git %s: %w", strings.Join(args, " "), err)
	}
	return strings.TrimSpace(string(out)), nil
}
