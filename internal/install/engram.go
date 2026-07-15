package install

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
)

const (
	engramModule  = "github.com/Gentleman-Programming/engram/cmd/engram"
	engramVersion = "v1.19.0"
)

func InstallEngram(runner Runner) (string, error) {
	goCommand, err := exec.LookPath("go")
	if err != nil {
		if !runner.DryRun {
			return "", fmt.Errorf("Engram requires Go 1.24 or newer: %w", err)
		}
		goCommand = "go"
	}
	configDir, err := os.UserConfigDir()
	if err != nil {
		return "", fmt.Errorf("resolve user config directory: %w", err)
	}
	binDir := filepath.Join(configDir, "token-optimizer", "bin")
	name := "engram"
	if runtime.GOOS == "windows" {
		name += ".exe"
	}
	if !runner.DryRun {
		if err := os.MkdirAll(binDir, 0o755); err != nil {
			return "", fmt.Errorf("create Engram binary directory: %w", err)
		}
	}
	if err := runner.RunWithEnv(map[string]string{"GOBIN": binDir}, goCommand, "install", engramModule+"@"+engramVersion); err != nil {
		return "", err
	}
	return filepath.Join(binDir, name), nil
}
