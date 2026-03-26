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
}

type Config struct {
	User        string
	WorkDir     string
	ProjectName string
}

type Prompter interface {
	Select(prompt string, options []string) (int, error)
	Input(prompt string) (string, error)
}

func Run(config Config, worktrees []Worktree, prompter Prompter) (result string, err error) {
	if len(worktrees) == 0 {
		return createNewWorktree(config, prompter)
	}

	options := []string{"Create new worktree"}
	for _, wt := range worktrees {
		options = append(options, fmt.Sprintf("%s (%s)", wt.Branch, wt.Path))
	}

	selected, err := prompter.Select("Select worktree:", options)
	if err != nil {
		return "", err
	}

	if selected == 0 {
		return createNewWorktree(config, prompter)
	}

	wt := worktrees[selected-1]
	return fmt.Sprintf(`cd "%s"`, wt.Path), nil
}

var validFeatureName = regexp.MustCompile(`^[a-zA-Z0-9]+(-[a-zA-Z0-9]+)*$`)

func createNewWorktree(config Config, prompter Prompter) (result string, err error) {
	feature, err := prompter.Input("Feature name (alphanumeric and hyphens only):")
	if err != nil {
		return "", err
	}
	if !validFeatureName.MatchString(feature) {
		return "", fmt.Errorf("invalid feature name %q: must contain only alphanumerics and hyphens", feature)
	}
	branch := filepath.Join(config.User, feature)
	path := filepath.Join(config.WorkDir, config.ProjectName, feature)
	return fmt.Sprintf(`git branch %s; git worktree add "%s" %s; cd "%s"`, branch, path, branch, path), nil
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
	return results
}
