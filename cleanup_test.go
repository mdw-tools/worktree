package worktree

import (
	"errors"
	"reflect"
	"testing"
)

type fakeConfirmer struct {
	answers []bool
	err     error
	prompts []string
}

func (this *fakeConfirmer) Confirm(_, description string) (bool, error) {
	i := len(this.prompts)
	this.prompts = append(this.prompts, description)
	if this.err != nil {
		return false, this.err
	}
	return i < len(this.answers) && this.answers[i], nil
}

type fakeRunner struct {
	commands []Command
	failOn   string
}

func (this *fakeRunner) Run(command Command) error {
	this.commands = append(this.commands, command)
	if this.failOn != "" && command.String() == this.failOn {
		return errors.New("boom")
	}
	return nil
}

func TestMergedWorktrees(t *testing.T) {
	worktrees := []Worktree{
		{Path: "/Users/mike/src/project", Branch: "main"},
		{Path: "/Users/mike/work/project/feature-one", Branch: "mikewhat/feature-one"},
		{Path: "/Users/mike/work/project/feature-two", Branch: "mikewhat/feature-two", Merged: true},
	}

	results := MergedWorktrees(worktrees)

	expected := []Worktree{
		{Path: "/Users/mike/work/project/feature-two", Branch: "mikewhat/feature-two", Merged: true},
	}
	if !reflect.DeepEqual(results, expected) {
		t.Errorf("expected %+v, got %+v", expected, results)
	}
}

func TestRepositories(t *testing.T) {
	dirs := []string{
		"/Users/mike/work/alpha/feature-one",
		"/Users/mike/work/alpha/feature-two",
		"/Users/mike/work/beta/feature-one",
		"/Users/mike/work/beta/not-a-repo",
		"/Users/mike/work/gamma/feature-one",
	}
	gitDirs := map[string]string{
		"/Users/mike/work/alpha/feature-one": "/Users/mike/src/alpha/.git",
		"/Users/mike/work/alpha/feature-two": "/Users/mike/src/alpha/.git",
		"/Users/mike/work/beta/feature-one":  "/Users/mike/src/beta/.git",
		"/Users/mike/work/gamma/feature-one": "/Users/mike/src/alpha/.git", // same repo, different project folder
	}
	gitDir := func(dir string) (string, error) {
		if repo, ok := gitDirs[dir]; ok {
			return repo, nil
		}
		return "", errors.New("not a git repository")
	}

	results := Repositories(dirs, gitDir)

	expected := []string{
		"/Users/mike/work/alpha/feature-one",
		"/Users/mike/work/beta/feature-one",
	}
	if !reflect.DeepEqual(results, expected) {
		t.Errorf("expected %+v, got %+v", expected, results)
	}
}

func TestCleanUp(t *testing.T) {
	candidates := []Worktree{
		{Path: "/Users/mike/work/project/feature-one", Branch: "mikewhat/feature-one"},
		{Path: "/Users/mike/work/project/feature-two", Branch: "mikewhat/feature-two"},
	}
	confirmer := &fakeConfirmer{answers: []bool{false, true}}
	runner := &fakeRunner{}

	failures, err := CleanUp("/Users/mike/src/project", candidates, confirmer, runner.Run)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(failures) != 0 {
		t.Errorf("expected no failures, got %v", failures)
	}
	expectedPrompts := []string{
		"mikewhat/feature-one (/Users/mike/work/project/feature-one)",
		"mikewhat/feature-two (/Users/mike/work/project/feature-two)",
	}
	if !reflect.DeepEqual(confirmer.prompts, expectedPrompts) {
		t.Errorf("expected prompts %+v, got %+v", expectedPrompts, confirmer.prompts)
	}
	expectedCommands := []Command{
		{Name: "git", Args: []string{"-C", "/Users/mike/src/project", "worktree", "remove", "/Users/mike/work/project/feature-two"}},
		{Name: "git", Args: []string{"-C", "/Users/mike/src/project", "branch", "-d", "mikewhat/feature-two"}},
	}
	if !reflect.DeepEqual(runner.commands, expectedCommands) {
		t.Errorf("expected commands %+v, got %+v", expectedCommands, runner.commands)
	}
}

func TestCleanUpContinuesAfterFailure(t *testing.T) {
	candidates := []Worktree{
		{Path: "/Users/mike/work/project/feature-one", Branch: "mikewhat/feature-one"},
		{Path: "/Users/mike/work/project/feature-two", Branch: "mikewhat/feature-two"},
	}
	confirmer := &fakeConfirmer{answers: []bool{true, true}}
	runner := &fakeRunner{failOn: "git -C /Users/mike/src/project worktree remove /Users/mike/work/project/feature-one"}

	failures, err := CleanUp("/Users/mike/src/project", candidates, confirmer, runner.Run)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(failures) != 1 {
		t.Fatalf("expected 1 failure, got %v", failures)
	}
	expectedCommands := []Command{
		{Name: "git", Args: []string{"-C", "/Users/mike/src/project", "worktree", "remove", "/Users/mike/work/project/feature-one"}},
		// branch deletion for feature-one is skipped because its worktree removal failed
		{Name: "git", Args: []string{"-C", "/Users/mike/src/project", "worktree", "remove", "/Users/mike/work/project/feature-two"}},
		{Name: "git", Args: []string{"-C", "/Users/mike/src/project", "branch", "-d", "mikewhat/feature-two"}},
	}
	if !reflect.DeepEqual(runner.commands, expectedCommands) {
		t.Errorf("expected commands %+v, got %+v", expectedCommands, runner.commands)
	}
}

func TestCleanUpAbortsOnPromptError(t *testing.T) {
	candidates := []Worktree{
		{Path: "/Users/mike/work/project/feature-one", Branch: "mikewhat/feature-one"},
		{Path: "/Users/mike/work/project/feature-two", Branch: "mikewhat/feature-two"},
	}
	confirmer := &fakeConfirmer{err: errCanceled}
	runner := &fakeRunner{}

	_, err := CleanUp("/Users/mike/src/project", candidates, confirmer, runner.Run)

	if !errors.Is(err, errCanceled) {
		t.Errorf("expected %v, got %v", errCanceled, err)
	}
	if len(confirmer.prompts) != 1 {
		t.Errorf("expected 1 prompt, got %d", len(confirmer.prompts))
	}
	if len(runner.commands) != 0 {
		t.Errorf("expected no commands, got %+v", runner.commands)
	}
}
