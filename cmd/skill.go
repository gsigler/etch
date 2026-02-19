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

func skillCmd() *cli.Command {
	return &cli.Command{
		Name:  "skill",
		Usage: "Manage etch Claude Code skills",
		Subcommands: []*cli.Command{
			skillInstallCmd(),
		},
	}
}

func skillInstallCmd() *cli.Command {
	return &cli.Command{
		Name:  "install",
		Usage: "Install or update etch skills in the current project",
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

type skillDef struct {
	name    string
	subPath string
	content string
}

var skills = []skillDef{
	{"etch-plan", "skills/etch-plan/SKILL.md", ""},
	{"etch", "skills/etch/SKILL.md", ""},
}

func init() {
	skills[0].content = skill.Content
	skills[1].content = skill.EtchContent
}

func runSkillInstall() error {
	claudeDir, err := resolveSkillDir()
	if err != nil {
		return err
	}

	for _, s := range skills {
		dest := filepath.Join(claudeDir, s.subPath)
		if err := writeSkillFile(dest, s.content); err != nil {
			return err
		}
		fmt.Printf("✓ Installed %s skill to %s\n", s.name, dest)
	}

	return nil
}

// installSkillsTo installs all skills into the given .claude directory
// without interactive prompts. Used by etch init.
func installSkillsTo(claudeDir string) error {
	for _, s := range skills {
		dest := filepath.Join(claudeDir, s.subPath)
		if err := writeSkillFile(dest, s.content); err != nil {
			return err
		}
	}
	return nil
}

func writeSkillFile(dest, content string) error {
	dir := filepath.Dir(dest)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return etcherr.WrapIO(fmt.Sprintf("creating directory %s", dir), err).
			WithHint("check file permissions in the current directory")
	}

	if err := os.WriteFile(dest, []byte(content), 0o644); err != nil {
		return etcherr.WrapIO(fmt.Sprintf("writing %s", dest), err).
			WithHint("check file permissions in the current directory")
	}

	return nil
}
