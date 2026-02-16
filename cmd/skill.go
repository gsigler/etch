package cmd

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	etcherr "github.com/gsigler/etch/internal/errors"
	"github.com/gsigler/etch/internal/skill"
	"github.com/urfave/cli/v2"
)

const skillSubPath = "skills/etch-plan/SKILL.md"

func skillCmd() *cli.Command {
	return &cli.Command{
		Name:  "skill",
		Usage: "Manage the etch-plan Claude Code skill",
		Subcommands: []*cli.Command{
			skillInstallCmd(),
		},
	}
}

func skillInstallCmd() *cli.Command {
	return &cli.Command{
		Name:  "install",
		Usage: "Install or update the etch-plan skill in the current project",
		Action: func(c *cli.Context) error {
			return runSkillInstall()
		},
	}
}

// findClaudeDir walks up from the current directory looking for an existing
// .claude directory. Returns the absolute path if found, or empty string.
func findClaudeDir() string {
	dir, err := os.Getwd()
	if err != nil {
		return ""
	}

	for {
		candidate := filepath.Join(dir, ".claude")
		if info, err := os.Stat(candidate); err == nil && info.IsDir() {
			return candidate
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	return ""
}

// resolveSkillDir determines where to install the skill file. It searches for
// an existing .claude directory first, and falls back to asking the user.
// Returns the absolute path to the .claude directory.
func resolveSkillDir() (string, error) {
	if found := findClaudeDir(); found != "" {
		return found, nil
	}

	cwd, err := os.Getwd()
	if err != nil {
		return "", etcherr.WrapIO("getting current directory", err)
	}
	defaultDir := filepath.Join(cwd, ".claude")

	fmt.Printf("No .claude directory found. Create one at %s? (Y/n) ", defaultDir)
	reader := bufio.NewReader(os.Stdin)
	answer, _ := reader.ReadString('\n')
	answer = strings.TrimSpace(strings.ToLower(answer))

	if answer != "" && answer != "y" && answer != "yes" {
		return "", etcherr.IO("skill installation cancelled").
			WithHint("create a .claude directory first, or run from a project that has one")
	}

	return defaultDir, nil
}

func skillFilePath(claudeDir string) string {
	return filepath.Join(claudeDir, skillSubPath)
}

func runSkillInstall() error {
	claudeDir, err := resolveSkillDir()
	if err != nil {
		return err
	}

	dest := skillFilePath(claudeDir)
	if err := writeSkillFile(dest); err != nil {
		return err
	}

	fmt.Printf("✓ Installed etch-plan skill to %s\n", dest)
	return nil
}

func writeSkillFile(dest string) error {
	dir := filepath.Dir(dest)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return etcherr.WrapIO(fmt.Sprintf("creating directory %s", dir), err).
			WithHint("check file permissions in the current directory")
	}

	if err := os.WriteFile(dest, []byte(skill.Content), 0o644); err != nil {
		return etcherr.WrapIO(fmt.Sprintf("writing %s", dest), err).
			WithHint("check file permissions in the current directory")
	}

	return nil
}
