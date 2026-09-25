package worktree

import (
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

func Run(config Config, worktrees []Worktree, prompter Prompter) (result Plan, err error) {
	if len(worktrees) <= 1 {
		return createNewWorktree(config, prompter)
	}

	options := []string{"Create new worktree"}
	for _, wt := range worktrees {
		options = append(options, wt.String())
	}
	options = append(options, "Delete a worktree")

	selected, err := prompter.Select("Select worktree:", options)
	if err != nil {
		return Plan{}, err
	}

	if selected == 0 {
		return createNewWorktree(config, prompter)
	}
	if selected == len(worktrees)+1 {
		return deleteWorktree(worktrees, prompter)
	}

	wt := worktrees[selected-1]
	return Plan{Dir: wt.Path}, nil
}

func deleteWorktree(worktrees []Worktree, prompter Prompter) (result Plan, err error) {
	candidates := worktrees[1:] // exclude the main worktree
	options := make([]string, len(candidates))
	for i, wt := range candidates {
		options[i] = wt.String()
	}

	selected, err := prompter.Select("Delete which worktree?", options)
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
