package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/mdw-tools/worktree"
)

var Version = "dev"

func main() {
	flags := flag.NewFlagSet(fmt.Sprintf("%s @ %s", filepath.Base(os.Args[0]), Version), flag.ExitOnError)
	user := flags.String("user", "mikewhat", "User branch prefix")
	workDir := flags.String("workdir", filepath.Join(os.Getenv("CODEPATH"), "work"), "Working path prefix for new worktrees")
	flags.Usage = func() {
		_, _ = fmt.Fprintf(flags.Output(), "Usage of %s:\n", flags.Name())
		_, _ = fmt.Fprintf(flags.Output(), "%s [flags]\n", filepath.Base(os.Args[0]))
		_, _ = fmt.Fprintln(flags.Output(), "Interactive git worktree manager. Emits shell commands to stdout.")
		flags.PrintDefaults()
	}
	_ = flags.Parse(os.Args[1:])

	repoRoot, err := gitOutput("rev-parse", "--show-toplevel")
	if err != nil {
		log.Fatal(err)
	}

	porcelain, err := gitOutput("worktree", "list", "--porcelain")
	if err != nil {
		log.Fatal(err)
	}

	worktrees := worktree.ParsePorcelain(porcelain)
	if len(worktrees) > 0 && worktrees[0].Branch != "" {
		primary := worktrees[0]
		merged, err := gitOutput("-C", primary.Path, "branch", "--merged", primary.Branch, "--format=%(refname:short)")
		if err != nil {
			log.Fatal(err)
		}
		worktrees = worktree.MarkMerged(worktrees, merged)
	}

	config := worktree.Config{
		User:        *user,
		WorkDir:     *workDir,
		ProjectName: filepath.Base(repoRoot),
	}

	prompter := &huhPrompter{}
	plan, err := worktree.Run(config, worktrees, prompter)
	if err != nil {
		log.Fatal(err)
	}

	if len(plan.Commands) > 0 {
		ok, err := prompter.Confirm("Run these commands?", commandList(plan.Commands))
		if err != nil {
			log.Fatal(err)
		}
		if !ok {
			_, _ = fmt.Fprintln(os.Stderr, "aborted")
			return
		}
		for _, command := range plan.Commands {
			if err := runCommand(command); err != nil {
				log.Fatalf("%s: %v", command, err)
			}
		}
	}

	if plan.Dir != "" {
		_, _ = fmt.Fprintf(os.Stderr, "→ Entering %s (type `exit` to return)\n", plan.Dir)
		if err := spawnShell(plan.Dir); err != nil {
			log.Fatal(err)
		}
	}
}

func commandList(commands []worktree.Command) (result string) {
	lines := make([]string, len(commands))
	for i, command := range commands {
		lines[i] = command.String()
	}
	return strings.Join(lines, "\n")
}

func runCommand(command worktree.Command) error {
	cmd := exec.Command(command.Name, command.Args...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func spawnShell(dir string) error {
	shell := os.Getenv("SHELL")
	if shell == "" {
		shell = "/bin/bash"
	}
	// Start a login shell (-l) so it re-sources the user's profile and defines
	// any prompt helpers (functions aren't inherited by child processes), just
	// like opening a fresh terminal tab.
	cmd := exec.Command(shell, "-l")
	cmd.Dir = dir
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func gitOutput(args ...string) (string, error) {
	out, err := exec.Command("git", args...).Output()
	if err != nil {
		return "", fmt.Errorf("git %s: %w", strings.Join(args, " "), err)
	}
	return strings.TrimSpace(string(out)), nil
}
