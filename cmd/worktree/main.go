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

	config := worktree.Config{
		User:        *user,
		WorkDir:     *workDir,
		ProjectName: filepath.Base(repoRoot),
	}

	prompter := &huhPrompter{}
	result, err := worktree.Run(config, worktrees, prompter)
	if err != nil {
		log.Fatal(err)
	}
	_, _ = fmt.Fprintln(os.Stderr, "→ Copy the command below, or re-run piped to `bash`:")
	fmt.Print(result)
}

func gitOutput(args ...string) (string, error) {
	out, err := exec.Command("git", args...).Output()
	if err != nil {
		return "", fmt.Errorf("git %s: %w", strings.Join(args, " "), err)
	}
	return strings.TrimSpace(string(out)), nil
}
