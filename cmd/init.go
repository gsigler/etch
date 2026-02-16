package cmd

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	etcherr "github.com/gsigler/etch/internal/errors"
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
			return etcherr.WrapIO(fmt.Sprintf("creating directory %s", d), err).
				WithHint("check file permissions in the current directory")
		}
	}

	configPath := filepath.Join(root, "config.toml")
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		if err := os.WriteFile(configPath, []byte(defaultConfig), 0o644); err != nil {
			return etcherr.WrapIO("writing config", err)
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
		return etcherr.WrapIO("updating .gitignore", err)
	}

	fmt.Println("✓ Etch initialized!")
	fmt.Println()
	fmt.Println("Next steps:")
	fmt.Println("  etch plan \"describe your feature\"    Generate an implementation plan")
	fmt.Println("  etch review <plan>                   Review and refine with AI")
	fmt.Println("  etch context <task>                  Generate context prompt file for a task")
	fmt.Println("  etch status                          Check progress across all plans")

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
[api]
# model = "claude-sonnet-4-20250514"
# api_key = ""  # or set ANTHROPIC_API_KEY env var

# Plan defaults
[defaults]
# complexity_guide = "small = single focused session, medium = may need iteration, large = multiple sessions likely"
`
