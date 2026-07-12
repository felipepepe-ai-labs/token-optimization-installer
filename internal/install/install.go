package install

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
)

type Plan struct {
	CodebaseMemory bool
	CodebaseUI     bool
	RTK            bool
	Headroom       bool
	Engram         bool
	VSCode         bool
	SkillsHub      bool
	DocsAgent      bool
	Workspace      string
}

type Runner struct {
	DryRun bool
	Out    io.Writer
}

type pythonEnvironment struct {
	Python   string
	Codebase string
	Headroom string
	Temp     string
}

func (r Runner) Run(name string, args ...string) error {
	return r.RunWithEnv(nil, name, args...)
}

func (r Runner) RunWithEnv(environment map[string]string, name string, args ...string) error {
	fmt.Fprintf(r.Out, "→ %s %v\n", name, args)
	if r.DryRun {
		return nil
	}
	cmd := exec.Command(name, args...)
	cmd.Env = os.Environ()
	for key, value := range environment {
		cmd.Env = append(cmd.Env, key+"="+value)
	}
	cmd.Stdout, cmd.Stderr, cmd.Stdin = r.Out, r.Out, os.Stdin
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("%s failed: %w", name, err)
	}
	return nil
}

func Execute(plan Plan, runner Runner) error {
	pythonEnv := pythonEnvironment{}
	engramCommand := ""
	if plan.CodebaseMemory || plan.CodebaseUI || plan.Headroom {
		var err error
		pythonEnv, err = ensurePythonEnvironment(runner)
		if err != nil {
			return err
		}
	}
	installCodebase := plan.CodebaseMemory || (plan.CodebaseUI && !fileExists(pythonEnv.Codebase))
	if installCodebase {
		if err := runner.RunWithEnv(pythonEnv.tempEnvironment(), pythonEnv.Python, "-m", "pip", "install", "--upgrade", "codebase-memory-mcp"); err != nil {
			return err
		}
		if existing, err := exec.LookPath("codebase-memory-mcp"); err == nil && !runner.DryRun {
			fmt.Fprintf(runner.Out, "✓ codebase-memory-mcp ya está instalado en %s\n", existing)
		} else if err := runner.RunWithEnv(pythonEnv.tempEnvironment(), pythonEnv.Codebase, "install", "-y"); err != nil {
			return err
		}
	}
	if plan.RTK {
		if _, err := exec.LookPath("cargo"); err != nil && !runner.DryRun {
			return fmt.Errorf("RTK requires Cargo/Rust: %w", err)
		}
		if err := runner.Run("cargo", "install", "--git", "https://github.com/rtk-ai/rtk", "--locked"); err != nil {
			return err
		}
		if err := runner.Run("rtk", "gain"); err != nil {
			return fmt.Errorf("RTK verification failed (possible name collision): %w", err)
		}
		if err := runner.Run("rtk", "init", "-g", "--auto-patch"); err != nil {
			return fmt.Errorf("configure RTK hooks for Copilot/VS Code: %w", err)
		}
	}
	if plan.Headroom {
		if err := runner.RunWithEnv(pythonEnv.tempEnvironment(), pythonEnv.Python, "-m", "pip", "install", "--upgrade", "headroom-ai[mcp]"); err != nil {
			return err
		}
	}
	if plan.Engram {
		var err error
		engramCommand, err = InstallEngram(runner)
		if err != nil {
			return err
		}
	}
	if plan.SkillsHub {
		if err := InstallSkillsHub(runner); err != nil {
			return err
		}
	}
	if plan.DocsAgent {
		if err := InstallDocsAgent(runner); err != nil {
			return err
		}
	}
	if plan.VSCode {
		if runner.DryRun {
			fmt.Fprintf(runner.Out, "→ configure %s\n", filepath.Join(plan.Workspace, ".vscode", "mcp.json"))
			return nil
		}
		codebaseCommand, headroomCommand := "", ""
		if plan.CodebaseMemory || plan.CodebaseUI {
			codebaseCommand = pythonEnv.Codebase
		}
		if plan.Headroom {
			headroomCommand = pythonEnv.Headroom
		}
		return ConfigureVSCode(plan.Workspace, codebaseCommand, plan.CodebaseUI, headroomCommand, engramCommand)
	}
	return nil
}

func findPython() (string, error) {
	candidates := []string{"python3", "python"}
	if runtime.GOOS == "windows" {
		candidates = []string{"py", "python"}
	}
	for _, candidate := range candidates {
		if path, err := exec.LookPath(candidate); err == nil {
			return path, nil
		}
	}
	return "", fmt.Errorf("Python 3 with pip is required")
}

func ensurePythonEnvironment(runner Runner) (pythonEnvironment, error) {
	basePython, err := findPython()
	if err != nil {
		return pythonEnvironment{}, err
	}
	configDir, err := os.UserConfigDir()
	if err != nil {
		return pythonEnvironment{}, fmt.Errorf("resolve user config directory: %w", err)
	}
	root := filepath.Join(configDir, "token-optimizer", "python")
	binDir := filepath.Join(root, "bin")
	pythonName := "python"
	codebaseName := "codebase-memory-mcp"
	headroomName := "headroom"
	if runtime.GOOS == "windows" {
		binDir = filepath.Join(root, "Scripts")
		pythonName = "python.exe"
		codebaseName += ".exe"
		headroomName += ".exe"
	}
	environment := pythonEnvironment{
		Python:   filepath.Join(binDir, pythonName),
		Codebase: filepath.Join(binDir, codebaseName),
		Headroom: filepath.Join(binDir, headroomName),
		Temp:     filepath.Join(configDir, "token-optimizer", "tmp"),
	}
	if _, err := os.Stat(environment.Python); os.IsNotExist(err) {
		if err := runner.Run(basePython, "-m", "venv", root); err != nil {
			return pythonEnvironment{}, fmt.Errorf("create private Python environment (install python3-venv if missing): %w", err)
		}
	}
	if !runner.DryRun {
		if err := os.MkdirAll(environment.Temp, 0o755); err != nil {
			return pythonEnvironment{}, fmt.Errorf("create private temporary directory: %w", err)
		}
	}
	return environment, nil
}

func (environment pythonEnvironment) tempEnvironment() map[string]string {
	return map[string]string{
		"TMPDIR": environment.Temp,
		"TEMP":   environment.Temp,
		"TMP":    environment.Temp,
	}
}

func ConfigureVSCode(workspace, codebaseCommand string, codebaseUI bool, headroomCommand, engramCommand string) error {
	path := filepath.Join(workspace, ".vscode", "mcp.json")
	config := map[string]any{}
	if data, err := os.ReadFile(path); err == nil && len(data) > 0 {
		if err := json.Unmarshal(data, &config); err != nil {
			return fmt.Errorf("parse %s: %w", path, err)
		}
	} else if err != nil && !os.IsNotExist(err) {
		return err
	}
	servers, ok := config["servers"].(map[string]any)
	if !ok {
		if _, exists := config["servers"]; exists {
			return fmt.Errorf("servers must be an object in %s", path)
		}
		servers = map[string]any{}
		config["servers"] = servers
	}
	if codebaseCommand != "" {
		args := []string{}
		if codebaseUI {
			args = []string{"--ui=true", "--port=9749"}
		}
		servers["codebase-memory-mcp"] = map[string]any{
			"type": "stdio", "command": codebaseCommand, "args": args,
		}
	}
	if headroomCommand != "" {
		servers["headroom"] = map[string]any{
			"type": "stdio", "command": headroomCommand, "args": []string{"mcp", "serve"},
		}
	}
	if engramCommand != "" {
		servers["engram"] = map[string]any{
			"type": "stdio", "command": engramCommand, "args": []string{"mcp", "--tools=agent"},
		}
	}
	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, append(data, '\n'), 0o644)
}
