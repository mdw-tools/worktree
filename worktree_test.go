package worktree

import (
	"errors"
	"testing"
)

type fakePrompter struct {
	selectIndices []int
	selectErrs    []error
	selectCall    int
	inputText     string
	inputErr      error
}

func (this *fakePrompter) Select(_ string, _ []string) (int, error) {
	i := this.selectCall
	this.selectCall++
	var idx int
	if i < len(this.selectIndices) {
		idx = this.selectIndices[i]
	}
	var err error
	if i < len(this.selectErrs) {
		err = this.selectErrs[i]
	}
	return idx, err
}

func (this *fakePrompter) Input(_ string) (string, error) {
	return this.inputText, this.inputErr
}

var errCanceled = errors.New("canceled")

func TestParsePorcelain(t *testing.T) {
	input := `worktree /Users/mike/src/project
HEAD abc123def456
branch refs/heads/main

worktree /Users/mike/work/project/feature-one
HEAD 789abc012def
branch refs/heads/mikewhat/feature-one

`
	results := ParsePorcelain(input)

	if len(results) != 2 {
		t.Fatalf("expected 2 worktrees, got %d", len(results))
	}
	if results[0].Path != "/Users/mike/src/project" {
		t.Errorf("expected path /Users/mike/src/project, got %s", results[0].Path)
	}
	if results[0].Branch != "main" {
		t.Errorf("expected branch main, got %s", results[0].Branch)
	}
	if results[1].Path != "/Users/mike/work/project/feature-one" {
		t.Errorf("expected path /Users/mike/work/project/feature-one, got %s", results[1].Path)
	}
	if results[1].Branch != "mikewhat/feature-one" {
		t.Errorf("expected branch mikewhat/feature-one, got %s", results[1].Branch)
	}
}

func TestParsePorcelainNoTrailingNewline(t *testing.T) {
	input := "worktree /Users/mike/src/project\nHEAD abc123\nbranch refs/heads/main\n\nworktree /Users/mike/work/project/feature-one\nHEAD 789abc\nbranch refs/heads/mikewhat/feature-one"
	results := ParsePorcelain(input)

	if len(results) != 2 {
		t.Fatalf("expected 2 worktrees, got %d", len(results))
	}
	if results[1].Branch != "mikewhat/feature-one" {
		t.Errorf("expected branch mikewhat/feature-one, got %s", results[1].Branch)
	}
}

func TestSelectExistingWorktree(t *testing.T) {
	worktrees := []Worktree{
		{Path: "/Users/mike/src/project", Branch: "main"},
		{Path: "/Users/mike/work/project/feature-one", Branch: "mikewhat/feature-one"},
	}
	prompter := &fakePrompter{selectIndices: []int{2}} // index 2 = second worktree (0 is "Create new")
	config := Config{
		User:        "mikewhat",
		WorkDir:     "/Users/mike/work",
		ProjectName: "project",
	}

	result, err := Run(config, worktrees, prompter)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	expected := `cd "/Users/mike/work/project/feature-one"`
	if result != expected {
		t.Errorf("expected %q, got %q", expected, result)
	}
}

func TestCreateNewWorktree(t *testing.T) {
	worktrees := []Worktree{
		{Path: "/Users/mike/src/project", Branch: "main"},
	}
	prompter := &fakePrompter{
		selectIndices: []int{0}, // "Create new worktree"
		inputText:     "my-feature",
	}
	config := Config{
		User:        "mikewhat",
		WorkDir:     "/Users/mike/work",
		ProjectName: "project",
	}

	result, err := Run(config, worktrees, prompter)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	expected := `git branch mikewhat/my-feature; git worktree add "/Users/mike/work/project/my-feature" mikewhat/my-feature; cd "/Users/mike/work/project/my-feature"`
	if result != expected {
		t.Errorf("expected %q, got %q", expected, result)
	}
}

func TestInvalidFeatureNames(t *testing.T) {
	config := Config{
		User:        "mikewhat",
		WorkDir:     "/Users/mike/work",
		ProjectName: "project",
	}
	worktrees := []Worktree{
		{Path: "/Users/mike/src/project", Branch: "main"},
	}

	invalid := []string{"has space", "special!char", "under_score", "dot.name", "slash/name", ""}
	for _, name := range invalid {
		prompter := &fakePrompter{selectIndices: []int{0}, inputText: name}
		_, err := Run(config, worktrees, prompter)
		if err == nil {
			t.Errorf("expected error for feature name %q, got nil", name)
		}
	}
}

func TestNoWorktreesSkipsMenu(t *testing.T) {
	prompter := &fakePrompter{
		selectIndices: []int{-1},
		selectErrs:    []error{errCanceled}, // Select should NOT be called
		inputText:     "new-feature",
	}
	config := Config{
		User:        "mikewhat",
		WorkDir:     "/Users/mike/work",
		ProjectName: "project",
	}

	result, err := Run(config, nil, prompter)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	expected := `git branch mikewhat/new-feature; git worktree add "/Users/mike/work/project/new-feature" mikewhat/new-feature; cd "/Users/mike/work/project/new-feature"`
	if result != expected {
		t.Errorf("expected %q, got %q", expected, result)
	}
}

func TestDeleteWorktree(t *testing.T) {
	worktrees := []Worktree{
		{Path: "/Users/mike/src/project", Branch: "main"},
		{Path: "/Users/mike/work/project/feature-one", Branch: "mikewhat/feature-one"},
	}
	prompter := &fakePrompter{
		selectIndices: []int{3, 0}, // 3 = "Delete a worktree" (last), 0 = feature-one (main excluded from sub-menu)
	}
	config := Config{
		User:        "mikewhat",
		WorkDir:     "/Users/mike/work",
		ProjectName: "project",
	}

	result, err := Run(config, worktrees, prompter)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	expected := `git worktree remove "/Users/mike/work/project/feature-one"; git branch -d mikewhat/feature-one`
	if result != expected {
		t.Errorf("expected %q, got %q", expected, result)
	}
}

func TestOnlyMainWorktreeSkipsToCreate(t *testing.T) {
	worktrees := []Worktree{
		{Path: "/Users/mike/src/project", Branch: "main"},
	}
	prompter := &fakePrompter{
		selectErrs: []error{errCanceled}, // Select should NOT be called
		inputText:  "new-feature",
	}
	config := Config{
		User:        "mikewhat",
		WorkDir:     "/Users/mike/work",
		ProjectName: "project",
	}

	result, err := Run(config, worktrees, prompter)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	expected := `git branch mikewhat/new-feature; git worktree add "/Users/mike/work/project/new-feature" mikewhat/new-feature; cd "/Users/mike/work/project/new-feature"`
	if result != expected {
		t.Errorf("expected %q, got %q", expected, result)
	}
}
