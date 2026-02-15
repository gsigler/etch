package cmd

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/urfave/cli/v2"
)

func initCmd() *cli.Command {
	return &cli.Command{
		Name:  "init",
		Usage: "Initialize etch in the current project",
		Action: func(c *cli.Context) error {
			return runInit()
		},
	}
}

func runInit() error {
	root := ".etch"

	dirs := []string{
		filepath.Join(root, "plans"),
		filepath.Join(root, "progress"),
		filepath.Join(root, "context"),
		filepath.Join(root, "backups"),
	}

	for _, d := range dirs {
		if err := os.MkdirAll(d, 0o755); err != nil {
			return fmt.Errorf("creating directory %s: %w", d, err)
		}
	}

	configPath := filepath.Join(root, "config.toml")
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		if err := os.WriteFile(configPath, []byte(defaultConfig), 0o644); err != nil {
			return fmt.Errorf("writing config: %w", err)
		}
	}

	// Ask about git tracking for progress files
	trackProgress := askYesNo("Track progress files in git? (y/N)")

	// Build gitignore entries
	var ignoreLines []string
	if !trackProgress {
		ignoreLines = append(ignoreLines, ".etch/progress/")
	}
	ignoreLines = append(ignoreLines,
		".etch/backups/",
		".etch/context/",
		".etch/config.toml",
	)

	if err := appendGitignore(ignoreLines); err != nil {
		return fmt.Errorf("updating .gitignore: %w", err)
	}

	fmt.Println("Initialized etch project.")
	fmt.Println()
	fmt.Println("Quickstart:")
	fmt.Println("  etch plan <description>   Generate an implementation plan")
	fmt.Println("  etch review <plan>        Review a plan interactively")
	fmt.Println("  etch status               Show current progress")

	return nil
}

func askYesNo(prompt string) bool {
	fmt.Print(prompt + " ")
	reader := bufio.NewReader(os.Stdin)
	answer, _ := reader.ReadString('\n')
	answer = strings.TrimSpace(strings.ToLower(answer))
	return answer == "y" || answer == "yes"
}

func appendGitignore(lines []string) error {
	const gitignorePath = ".gitignore"

	existing := make(map[string]bool)
	if data, err := os.ReadFile(gitignorePath); err == nil {
		for _, line := range strings.Split(string(data), "\n") {
			existing[strings.TrimSpace(line)] = true
		}
	}

	var toAdd []string
	for _, line := range lines {
		if !existing[line] {
			toAdd = append(toAdd, line)
		}
	}

	if len(toAdd) == 0 {
		return nil
	}

	f, err := os.OpenFile(gitignorePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	defer f.Close()

	// Ensure we start on a new line
	if info, err := f.Stat(); err == nil && info.Size() > 0 {
		// Check if file ends with newline
		data, _ := os.ReadFile(gitignorePath)
		if len(data) > 0 && data[len(data)-1] != '\n' {
			fmt.Fprintln(f)
		}
	}

	for _, line := range toAdd {
		fmt.Fprintln(f, line)
	}

	return nil
}

const defaultConfig = `# Etch configuration
# See https://github.com/gsigler/etch for documentation

# AI provider settings
[ai]
# provider = "anthropic"
# model = "claude-sonnet-4-5-20250929"

# Plan defaults
[plan]
# max_features = 10
# max_tasks_per_feature = 10

# Context generation
[context]
# include_progress = true
# include_plan = true
`
