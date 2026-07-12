package install

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
)

const (
	docsAgentRepository = "https://github.com/FelipePepe/docs-agent-vscode.git"
	docsAgentVersion    = "v0.3.5"
)

func InstallDocsAgent(runner Runner) error {
	configDir, err := os.UserConfigDir()
	if err != nil {
		return fmt.Errorf("resolve user config directory: %w", err)
	}
	repoDir := filepath.Join(configDir, "token-optimizer", "docs-agent-vscode")
	if _, err := os.Stat(filepath.Join(repoDir, ".git")); os.IsNotExist(err) {
		if err := runner.Run("git", "clone", "--branch", docsAgentVersion, "--depth", "1", docsAgentRepository, repoDir); err != nil {
			return err
		}
	} else {
		if err := runner.Run("git", "-C", repoDir, "fetch", "--depth", "1", "origin", "tag", docsAgentVersion); err != nil {
			return err
		}
		if err := runner.Run("git", "-C", repoDir, "checkout", "--detach", docsAgentVersion); err != nil {
			return fmt.Errorf("docs-agent checkout has local changes or cannot select %s: %w", docsAgentVersion, err)
		}
	}
	if err := runner.Run("pnpm", "--dir", repoDir, "install", "--frozen-lockfile"); err != nil {
		return fmt.Errorf("install Docs Agent dependencies (Node 22+ and pnpm required): %w", err)
	}
	if err := runner.Run("pnpm", "--dir", repoDir, "run", "package"); err != nil {
		return fmt.Errorf("package Docs Agent (Windows C++ build tools may be required): %w", err)
	}
	vsix := filepath.Join(repoDir, "docs-agent-0.3.5.vsix")
	if !runner.DryRun {
		vsix, err = newestVSIX(repoDir)
		if err != nil {
			return err
		}
	}
	if err := runner.Run("code", "--install-extension", vsix, "--force"); err != nil {
		return fmt.Errorf("install Docs Agent in VS Code (ensure the code CLI is in PATH): %w", err)
	}
	return nil
}

func newestVSIX(directory string) (string, error) {
	matches, err := filepath.Glob(filepath.Join(directory, "docs-agent-*.vsix"))
	if err != nil {
		return "", err
	}
	if len(matches) == 0 {
		return "", fmt.Errorf("no docs-agent VSIX produced in %s", directory)
	}
	sort.Strings(matches)
	return matches[len(matches)-1], nil
}
