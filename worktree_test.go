package worktree

import (
	"errors"
	"reflect"
	"testing"
)

type fakePrompter struct {
	selectIndices []int
	selectErrs    []error
	selectCall    int
	selectPrompts []string
	selectOptions [][]string
	inputText     string
	inputErr      error
}

func (this *fakePrompter) Select(prompt string, options []string) (int, error) {
	this.selectPrompts = append(this.selectPrompts, prompt)
	this.selectOptions = append(this.selectOptions, options)
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

var testConfig = Config{
	User:        "mikewhat",
	WorkDir:     "/Users/mike/work",
	ProjectName: "project",
}

var testWorktrees = []Worktree{
	{Path: "/Users/mike/src/project", Branch: "main"},
	{Path: "/Users/mike/work/project/feature-one", Branch: "mikewhat/feature-one"},
}

func TestTopLevelMenu(t *testing.T) {
	prompter := &fakePrompter{selectErrs: []error{errCanceled}}

	_, err := Run(testConfig, testWorktrees, nil, prompter)

	if !errors.Is(err, errCanceled) {
		t.Errorf("expected %v, got %v", errCanceled, err)
	}
	expected := [][]string{
		{"Enter worktree", "Create new worktree", "Create worktree from branch", "Delete worktree"},
	}
	if !reflect.DeepEqual(prompter.selectOptions, expected) {
		t.Errorf("expected %+v, got %+v", expected, prompter.selectOptions)
	}
}

func TestTopLevelMenuWithOnlyMainWorktreeOmitsEnterAndDelete(t *testing.T) {
	for _, worktrees := range [][]Worktree{nil, testWorktrees[:1]} {
		prompter := &fakePrompter{selectErrs: []error{errCanceled}}

		_, err := Run(testConfig, worktrees, nil, prompter)

		if !errors.Is(err, errCanceled) {
			t.Errorf("expected %v, got %v", errCanceled, err)
		}
		expected := [][]string{
			{"Create new worktree", "Create worktree from branch"},
		}
		if !reflect.DeepEqual(prompter.selectOptions, expected) {
			t.Errorf("expected %+v, got %+v", expected, prompter.selectOptions)
		}
	}
}

func TestTopLevelMenuTitleCountsLinkedWorktrees(t *testing.T) {
	worktrees := []Worktree{
		{Path: "/Users/mike/src/project", Branch: "main"},
		{Path: "/Users/mike/work/project/feature-one", Branch: "mikewhat/feature-one"},
		{Path: "/Users/mike/work/project/feature-two", Branch: "mikewhat/feature-two"},
	}
	cases := []struct {
		worktrees []Worktree
		expected  string
	}{
		{worktrees: nil, expected: "project has 0 worktrees. What would you like to do?"},
		{worktrees: worktrees[:1], expected: "project has 0 worktrees. What would you like to do?"},
		{worktrees: worktrees[:2], expected: "project has 1 worktree. What would you like to do?"},
		{worktrees: worktrees, expected: "project has 2 worktrees. What would you like to do?"},
	}
	for _, c := range cases {
		prompter := &fakePrompter{selectErrs: []error{errCanceled}}

		_, _ = Run(testConfig, c.worktrees, nil, prompter)

		if prompter.selectPrompts[0] != c.expected {
			t.Errorf("expected %q, got %q", c.expected, prompter.selectPrompts[0])
		}
	}
}

func TestCreateWithOnlyMainWorktree(t *testing.T) {
	branches := []Branch{
		{Name: "mikewhat/feature-two"},
	}
	create := &fakePrompter{selectIndices: []int{0}, inputText: "my-feature"} // "Create new worktree"
	fromBranch := &fakePrompter{selectIndices: []int{1, 0}}                   // "Create worktree from branch", feature-two

	createResult, createErr := Run(testConfig, testWorktrees[:1], branches, create)
	fromBranchResult, fromBranchErr := Run(testConfig, testWorktrees[:1], branches, fromBranch)

	if createErr != nil || fromBranchErr != nil {
		t.Fatalf("unexpected errors: %v, %v", createErr, fromBranchErr)
	}
	if createResult.Dir != "/Users/mike/work/project/my-feature" {
		t.Errorf("expected new worktree, got %+v", createResult)
	}
	if fromBranchResult.Dir != "/Users/mike/work/project/feature-two" {
		t.Errorf("expected worktree from branch, got %+v", fromBranchResult)
	}
}

func TestEnterWorktree(t *testing.T) {
	prompter := &fakePrompter{selectIndices: []int{0, 1}} // "Enter worktree", feature-one

	result, err := Run(testConfig, testWorktrees, nil, prompter)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	expected := Plan{Dir: "/Users/mike/work/project/feature-one"}
	if !reflect.DeepEqual(result, expected) {
		t.Errorf("expected %+v, got %+v", expected, result)
	}
}

func TestEnterMainWorktree(t *testing.T) {
	prompter := &fakePrompter{selectIndices: []int{0, 0}} // "Enter worktree", main

	result, err := Run(testConfig, testWorktrees, nil, prompter)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	expected := Plan{Dir: "/Users/mike/src/project"}
	if !reflect.DeepEqual(result, expected) {
		t.Errorf("expected %+v, got %+v", expected, result)
	}
}

func TestCreateNewWorktree(t *testing.T) {
	prompter := &fakePrompter{
		selectIndices: []int{1}, // "Create new worktree"
		inputText:     "my-feature",
	}

	result, err := Run(testConfig, testWorktrees, nil, prompter)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	expected := Plan{
		Commands: []Command{
			{Name: "git", Args: []string{"branch", "mikewhat/my-feature"}},
			{Name: "git", Args: []string{"worktree", "add", "/Users/mike/work/project/my-feature", "mikewhat/my-feature"}},
		},
		Dir: "/Users/mike/work/project/my-feature",
	}
	if !reflect.DeepEqual(result, expected) {
		t.Errorf("expected %+v, got %+v", expected, result)
	}
}

func TestInvalidFeatureNames(t *testing.T) {
	invalid := []string{"has space", "special!char", "under_score", "dot.name", "slash/name", ""}
	for _, name := range invalid {
		prompter := &fakePrompter{selectIndices: []int{1}, inputText: name}
		_, err := Run(testConfig, testWorktrees, nil, prompter)
		if err == nil {
			t.Errorf("expected error for feature name %q, got nil", name)
		}
	}
}

func TestCreateWorktreeFromBranchOffersOnlyBranchesNotCheckedOut(t *testing.T) {
	branches := []Branch{
		{Name: "main"},
		{Name: "mikewhat/feature-one"},
		{Name: "mikewhat/feature-two"},
		{Name: "alice/fix", Remote: "origin"},
	}
	prompter := &fakePrompter{selectIndices: []int{2, 0}} // "Create worktree from branch", feature-two

	_, err := Run(testConfig, testWorktrees, branches, prompter)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	expected := []string{"mikewhat/feature-two", "origin/alice/fix"}
	if !reflect.DeepEqual(prompter.selectOptions[1], expected) {
		t.Errorf("expected %+v, got %+v", expected, prompter.selectOptions[1])
	}
}

func TestCreateWorktreeFromLocalBranch(t *testing.T) {
	branches := []Branch{
		{Name: "mikewhat/feature-two"},
	}
	prompter := &fakePrompter{selectIndices: []int{2, 0}} // "Create worktree from branch", feature-two

	result, err := Run(testConfig, testWorktrees, branches, prompter)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	expected := Plan{
		Commands: []Command{
			{Name: "git", Args: []string{"worktree", "add", "/Users/mike/work/project/feature-two", "mikewhat/feature-two"}},
		},
		Dir: "/Users/mike/work/project/feature-two",
	}
	if !reflect.DeepEqual(result, expected) {
		t.Errorf("expected %+v, got %+v", expected, result)
	}
}

func TestCreateWorktreeFromAnotherUsersBranch(t *testing.T) {
	branches := []Branch{
		{Name: "alice/fix/flaky-test"},
	}
	prompter := &fakePrompter{selectIndices: []int{2, 0}} // "Create worktree from branch", alice/fix/flaky-test

	result, err := Run(testConfig, testWorktrees, branches, prompter)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	expected := Plan{
		Commands: []Command{
			{Name: "git", Args: []string{"worktree", "add", "/Users/mike/work/project/alice-fix-flaky-test", "alice/fix/flaky-test"}},
		},
		Dir: "/Users/mike/work/project/alice-fix-flaky-test",
	}
	if !reflect.DeepEqual(result, expected) {
		t.Errorf("expected %+v, got %+v", expected, result)
	}
}

func TestCreateWorktreeFromRemoteBranch(t *testing.T) {
	branches := []Branch{
		{Name: "alice/fix", Remote: "origin"},
	}
	prompter := &fakePrompter{selectIndices: []int{2, 0}} // "Create worktree from branch", origin/alice/fix

	result, err := Run(testConfig, testWorktrees, branches, prompter)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	expected := Plan{
		Commands: []Command{
			{Name: "git", Args: []string{"worktree", "add", "--track", "-b", "alice/fix", "/Users/mike/work/project/alice-fix", "origin/alice/fix"}},
		},
		Dir: "/Users/mike/work/project/alice-fix",
	}
	if !reflect.DeepEqual(result, expected) {
		t.Errorf("expected %+v, got %+v", expected, result)
	}
}

func TestCreateWorktreeFromBranchWithNoCandidates(t *testing.T) {
	branches := []Branch{
		{Name: "main"},
		{Name: "mikewhat/feature-one"},
	}
	prompter := &fakePrompter{selectIndices: []int{2}} // "Create worktree from branch"

	_, err := Run(testConfig, testWorktrees, branches, prompter)

	if err == nil {
		t.Error("expected error, got nil")
	}
	if prompter.selectCall != 1 {
		t.Errorf("expected only the top-level menu, got %d selections", prompter.selectCall)
	}
}

func TestDeleteWorktree(t *testing.T) {
	prompter := &fakePrompter{
		selectIndices: []int{3, 0}, // "Delete worktree", feature-one (main excluded from sub-menu)
	}

	result, err := Run(testConfig, testWorktrees, nil, prompter)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	expected := Plan{
		Commands: []Command{
			{Name: "git", Args: []string{"worktree", "remove", "/Users/mike/work/project/feature-one"}},
			{Name: "git", Args: []string{"branch", "-d", "mikewhat/feature-one"}},
		},
	}
	if !reflect.DeepEqual(result, expected) {
		t.Errorf("expected %+v, got %+v", expected, result)
	}
}

func TestSubMenuCanceled(t *testing.T) {
	for _, option := range []int{0, 2, 3} {
		branches := []Branch{{Name: "mikewhat/feature-two"}}
		prompter := &fakePrompter{
			selectIndices: []int{option, 0},
			selectErrs:    []error{nil, errCanceled},
		}

		result, err := Run(testConfig, testWorktrees, branches, prompter)

		if !errors.Is(err, errCanceled) {
			t.Errorf("option %d: expected %v, got %v", option, errCanceled, err)
		}
		if !reflect.DeepEqual(result, Plan{}) {
			t.Errorf("option %d: expected empty plan, got %+v", option, result)
		}
	}
}

func TestParseBranches(t *testing.T) {
	input := `refs/heads/main
refs/heads/mikewhat/feature-one
refs/remotes/origin/HEAD
refs/remotes/origin/main
refs/remotes/origin/alice/fix
refs/remotes/upstream/bob/review
`
	results := ParseBranches(input)

	expected := []Branch{
		{Name: "main"},
		{Name: "mikewhat/feature-one"},
		{Name: "alice/fix", Remote: "origin"},
		{Name: "bob/review", Remote: "upstream"},
	}
	if !reflect.DeepEqual(results, expected) {
		t.Errorf("expected %+v, got %+v", expected, results)
	}
}

func TestMarkMerged(t *testing.T) {
	worktrees := []Worktree{
		{Path: "/Users/mike/src/project", Branch: "main"},
		{Path: "/Users/mike/work/project/feature-one", Branch: "mikewhat/feature-one"},
		{Path: "/Users/mike/work/project/feature-two", Branch: "mikewhat/feature-two"},
		{Path: "/Users/mike/work/project/detached"},
	}
	merged := "main\nmikewhat/feature-two\nsome-other-branch\n"

	results := MarkMerged(worktrees, merged)

	expected := []Worktree{
		{Path: "/Users/mike/src/project", Branch: "main"}, // the main worktree is never flagged
		{Path: "/Users/mike/work/project/feature-one", Branch: "mikewhat/feature-one"},
		{Path: "/Users/mike/work/project/feature-two", Branch: "mikewhat/feature-two", Merged: true},
		{Path: "/Users/mike/work/project/detached"},
	}
	if !reflect.DeepEqual(results, expected) {
		t.Errorf("expected %+v, got %+v", expected, results)
	}
}

func TestMergedWorktreesAreFlaggedInMenus(t *testing.T) {
	worktrees := []Worktree{
		{Path: "/Users/mike/src/project", Branch: "main"},
		{Path: "/Users/mike/work/project/feature-one", Branch: "mikewhat/feature-one"},
		{Path: "/Users/mike/work/project/feature-two", Branch: "mikewhat/feature-two", Merged: true},
	}
	enter := &fakePrompter{selectIndices: []int{0, 0}}  // "Enter worktree", main
	remove := &fakePrompter{selectIndices: []int{3, 0}} // "Delete worktree", feature-one

	_, enterErr := Run(testConfig, worktrees, nil, enter)
	_, removeErr := Run(testConfig, worktrees, nil, remove)

	if enterErr != nil || removeErr != nil {
		t.Fatalf("unexpected errors: %v, %v", enterErr, removeErr)
	}
	expectedEnter := []string{
		"main (/Users/mike/src/project)",
		"mikewhat/feature-one (/Users/mike/work/project/feature-one)",
		"mikewhat/feature-two (/Users/mike/work/project/feature-two) [merged]",
	}
	if !reflect.DeepEqual(enter.selectOptions[1], expectedEnter) {
		t.Errorf("expected %+v, got %+v", expectedEnter, enter.selectOptions[1])
	}
	expectedDelete := []string{
		"mikewhat/feature-one (/Users/mike/work/project/feature-one)",
		"mikewhat/feature-two (/Users/mike/work/project/feature-two) [merged]",
	}
	if !reflect.DeepEqual(remove.selectOptions[1], expectedDelete) {
		t.Errorf("expected %+v, got %+v", expectedDelete, remove.selectOptions[1])
	}
}
