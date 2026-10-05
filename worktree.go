package worktree

import (
	"errors"
	"fmt"
	"path/filepath"
	"regexp"
	"strings"
)

type Worktree struct {
	Path   string
	Branch string
	Merged bool // Branch has been merged into the main worktree's branch.
}

func (this Worktree) String() string {
	result := fmt.Sprintf("%s (%s)", this.Branch, this.Path)
	if this.Merged {
		result += " [merged]"
	}
	return result
}

type Config struct {
	User        string
	WorkDir     string
	ProjectName string
}

// Command is a single program invocation to be run via exec.Command.
type Command struct {
	Name string
	Args []string
}

func (this Command) String() string {
	return strings.TrimSpace(this.Name + " " + strings.Join(this.Args, " "))
}

// Plan describes the side effects to carry out: a sequence of commands to
// execute (after confirmation), and a directory to land in afterward. An empty
// Dir means "stay where you are".
type Plan struct {
	Commands []Command
	Dir      string
}

type Prompter interface {
	Select(prompt string, options []string) (int, error)
	Input(prompt string) (string, error)
}

// Branch is an existing branch from which a worktree may be created. Remote is
// empty for a local branch; otherwise the branch exists only on that remote.
type Branch struct {
	Name   string
	Remote string
}

func (this Branch) String() string {
	if this.Remote == "" {
		return this.Name
	}
	return this.Remote + "/" + this.Name
}

const (
	optionEnter            = "Enter worktree"
	optionCreate           = "Create new worktree"
	optionCreateFromBranch = "Create worktree from branch"
	optionDelete           = "Delete worktree"
)

// Run offers the top-level menu, whose title counts the linked worktrees (the
// main worktree isn't counted). Entering and deleting are only offered when at
// least one linked worktree exists.
func Run(config Config, worktrees []Worktree, branches []Branch, prompter Prompter) (result Plan, err error) {
	count := max(len(worktrees)-1, 0)
	options := []string{optionCreate, optionCreateFromBranch}
	if count > 0 {
		options = []string{optionEnter, optionCreate, optionCreateFromBranch, optionDelete}
	}
	noun := "worktrees"
	if count == 1 {
		noun = "worktree"
	}
	title := fmt.Sprintf("%s has %d %s. What would you like to do?", config.ProjectName, count, noun)

	selected, err := prompter.Select(title, options)
	if err != nil {
		return Plan{}, err
	}
	switch options[selected] {
	case optionEnter:
		return enterWorktree(worktrees, prompter)
	case optionCreateFromBranch:
		return createWorktreeFromBranch(config, worktrees, branches, prompter)
	case optionDelete:
		return deleteWorktree(worktrees, prompter)
	default:
		return createNewWorktree(config, prompter)
	}
}

func enterWorktree(worktrees []Worktree, prompter Prompter) (result Plan, err error) {
	selected, err := prompter.Select("Enter which worktree?", worktreeOptions(worktrees))
	if err != nil {
		return Plan{}, err
	}
	return Plan{Dir: worktrees[selected].Path}, nil
}

func deleteWorktree(worktrees []Worktree, prompter Prompter) (result Plan, err error) {
	candidates := worktrees[1:] // exclude the main worktree
	selected, err := prompter.Select("Delete which worktree?", worktreeOptions(candidates))
	if err != nil {
		return Plan{}, err
	}

	wt := candidates[selected]
	return Plan{
		Commands: []Command{
			{Name: "git", Args: []string{"worktree", "remove", wt.Path}},
			{Name: "git", Args: []string{"branch", "-d", wt.Branch}},
		},
	}, nil
}

func worktreeOptions(worktrees []Worktree) (results []string) {
	for _, wt := range worktrees {
		results = append(results, wt.String())
	}
	return results
}

var validFeatureName = regexp.MustCompile(`^[a-zA-Z0-9]+(-[a-zA-Z0-9]+)*$`)

func createNewWorktree(config Config, prompter Prompter) (result Plan, err error) {
	feature, err := prompter.Input("Feature name (alphanumeric and hyphens only):")
	if err != nil {
		return Plan{}, err
	}
	if !validFeatureName.MatchString(feature) {
		return Plan{}, fmt.Errorf("invalid feature name %q: must contain only alphanumerics and hyphens", feature)
	}
	branch := filepath.Join(config.User, feature)
	path := filepath.Join(config.WorkDir, config.ProjectName, feature)
	return Plan{
		Commands: []Command{
			{Name: "git", Args: []string{"branch", branch}},
			{Name: "git", Args: []string{"worktree", "add", path, branch}},
		},
		Dir: path,
	}, nil
}

// createWorktreeFromBranch offers every branch not already checked out in a
// worktree. A remote branch gets a new local branch of the same name that
// tracks it. The worktree's directory is named for the branch, minus the
// configured user prefix, with any remaining slashes turned into hyphens.
func createWorktreeFromBranch(config Config, worktrees []Worktree, branches []Branch, prompter Prompter) (result Plan, err error) {
	checkedOut := make(map[string]bool)
	for _, wt := range worktrees {
		checkedOut[wt.Branch] = true
	}
	var candidates []Branch
	var options []string
	for _, branch := range branches {
		if !checkedOut[branch.Name] {
			candidates = append(candidates, branch)
			options = append(options, branch.String())
		}
	}
	if len(candidates) == 0 {
		return Plan{}, errors.New("no branches available (every branch is already checked out in a worktree)")
	}

	selected, err := prompter.Select("Create worktree from which branch?", options)
	if err != nil {
		return Plan{}, err
	}

	branch := candidates[selected]
	name := strings.ReplaceAll(strings.TrimPrefix(branch.Name, config.User+"/"), "/", "-")
	path := filepath.Join(config.WorkDir, config.ProjectName, name)
	args := []string{"worktree", "add", path, branch.Name}
	if branch.Remote != "" {
		args = []string{"worktree", "add", "--track", "-b", branch.Name, path, branch.String()}
	}
	return Plan{
		Commands: []Command{
			{Name: "git", Args: args},
		},
		Dir: path,
	}, nil
}

func ParsePorcelain(output string) (results []Worktree) {
	var current Worktree
	for _, line := range strings.Split(output, "\n") {
		switch {
		case strings.HasPrefix(line, "worktree "):
			current = Worktree{Path: strings.TrimPrefix(line, "worktree ")}
		case strings.HasPrefix(line, "branch "):
			current.Branch = strings.TrimPrefix(line, "branch refs/heads/")
		case line == "":
			if current.Path != "" {
				results = append(results, current)
				current = Worktree{}
			}
		}
	}
	if current.Path != "" {
		results = append(results, current)
	}
	return results
}

// ParseBranches interprets the output of
// `git for-each-ref --format=%(refname) refs/heads refs/remotes`, yielding all
// local branches followed by each remote branch that has no local branch of
// the same name.
func ParseBranches(output string) (results []Branch) {
	local := make(map[string]bool)
	var remote []Branch
	for _, line := range strings.Split(output, "\n") {
		line = strings.TrimSpace(line)
		if name, ok := strings.CutPrefix(line, "refs/heads/"); ok {
			local[name] = true
			results = append(results, Branch{Name: name})
		} else if rest, ok := strings.CutPrefix(line, "refs/remotes/"); ok {
			remoteName, name, found := strings.Cut(rest, "/")
			if found && name != "HEAD" {
				remote = append(remote, Branch{Name: name, Remote: remoteName})
			}
		}
	}
	for _, branch := range remote {
		if !local[branch.Name] {
			results = append(results, branch)
		}
	}
	return results
}

// MarkMerged flags each non-main worktree whose branch appears in merged,
// which is expected to be the output of
// `git branch --merged <main-branch> --format=%(refname:short)`.
func MarkMerged(worktrees []Worktree, merged string) (results []Worktree) {
	branches := make(map[string]bool)
	for _, line := range strings.Split(merged, "\n") {
		if branch := strings.TrimSpace(line); branch != "" {
			branches[branch] = true
		}
	}
	for i, wt := range worktrees {
		wt.Merged = i > 0 && wt.Branch != "" && branches[wt.Branch]
		results = append(results, wt)
	}
	return results
}
