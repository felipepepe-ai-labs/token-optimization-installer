package install

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestEnsurePythonEnvironmentUsesPrivateVenv(t *testing.T) {
	config := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", config)
	var output bytes.Buffer

	environment, err := ensurePythonEnvironment(Runner{DryRun: true, Out: &output})
	if err != nil {
		t.Fatal(err)
	}
	wantRoot := filepath.Join(config, "token-optimizer", "python")
	if environment.Python != filepath.Join(wantRoot, "bin", "python") {
		t.Fatalf("python path = %q", environment.Python)
	}
	if environment.Temp != filepath.Join(config, "token-optimizer", "tmp") {
		t.Fatalf("temp path = %q", environment.Temp)
	}
	if !bytes.Contains(output.Bytes(), []byte("-m venv")) {
		t.Fatalf("venv creation not planned: %s", output.String())
	}
}

func TestInstallEngramUsesPrivateBinaryDirectory(t *testing.T) {
	config := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", config)
	var output bytes.Buffer

	command, err := InstallEngram(Runner{DryRun: true, Out: &output})
	if err != nil {
		t.Fatal(err)
	}
	want := filepath.Join(config, "token-optimizer", "bin", "engram")
	if command != want {
		t.Fatalf("Engram command = %q, want %q", command, want)
	}
	if !bytes.Contains(output.Bytes(), []byte(engramModule)) {
		t.Fatalf("Engram installation not planned: %s", output.String())
	}
}

func TestRTKInstallationConfiguresHooksWithoutPrompts(t *testing.T) {
	var output bytes.Buffer
	if err := Execute(Plan{RTK: true}, Runner{DryRun: true, Out: &output}); err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(output.Bytes(), []byte("rtk [init -g --auto-patch]")) {
		t.Fatalf("RTK hook configuration not planned: %s", output.String())
	}
}

func TestConfigureVSCodePreservesServers(t *testing.T) {
	root := t.TempDir()
	dir := filepath.Join(root, ".vscode")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	existing := []byte(`{"servers":{"existing":{"type":"http","url":"https://example.test"}},"inputs":[]}`)
	if err := os.WriteFile(filepath.Join(dir, "mcp.json"), existing, 0o644); err != nil {
		t.Fatal(err)
	}

	if err := ConfigureVSCode(root, "/private/bin/codebase-memory-mcp", true, "/private/bin/headroom", "/private/bin/engram"); err != nil {
		t.Fatal(err)
	}
	var config map[string]any
	data, _ := os.ReadFile(filepath.Join(dir, "mcp.json"))
	if err := json.Unmarshal(data, &config); err != nil {
		t.Fatal(err)
	}
	servers := config["servers"].(map[string]any)
	for _, name := range []string{"existing", "codebase-memory-mcp", "headroom", "engram"} {
		if _, ok := servers[name]; !ok {
			t.Fatalf("missing server %q", name)
		}
	}
	codebase := servers["codebase-memory-mcp"].(map[string]any)
	if codebase["command"] != "/private/bin/codebase-memory-mcp" {
		t.Fatalf("unexpected command: %v", codebase["command"])
	}
	codebaseArgs := codebase["args"].([]any)
	if len(codebaseArgs) != 2 || codebaseArgs[0] != "--ui=true" || codebaseArgs[1] != "--port=9749" {
		t.Fatalf("unexpected Codebase Memory UI args: %v", codebaseArgs)
	}
	engram := servers["engram"].(map[string]any)
	if engram["command"] != "/private/bin/engram" {
		t.Fatalf("unexpected Engram command: %v", engram["command"])
	}
	args := engram["args"].([]any)
	if len(args) != 2 || args[0] != "mcp" || args[1] != "--tools=agent" {
		t.Fatalf("unexpected Engram args: %v", args)
	}
}

func TestConfigureVSCodeIsIdempotent(t *testing.T) {
	root := t.TempDir()
	if err := ConfigureVSCode(root, "codebase", false, "headroom", "engram"); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(root, ".vscode", "mcp.json")
	first, _ := os.ReadFile(path)
	if err := ConfigureVSCode(root, "codebase", false, "headroom", "engram"); err != nil {
		t.Fatal(err)
	}
	second, _ := os.ReadFile(path)
	if string(first) != string(second) {
		t.Fatal("configuration changed on second run")
	}
}
