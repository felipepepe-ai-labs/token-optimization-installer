package install

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestSyncSkillsHubWindowsNative(t *testing.T) {
	repo, home := t.TempDir(), t.TempDir()
	manifest := skillsHubManifest{Apps: []skillsHubApp{{
		ID:          "copilot",
		InstallPath: map[string]string{"windows": `%USERPROFILE%/.copilot/skills`},
		DetectPaths: map[string][]string{"windows": {`%USERPROFILE%/.copilot`}},
		Sources:     []string{"skills/common"},
		ConfigFiles: []skillsHubConfig{{
			Source: "behavior/copilot.md", Target: map[string]string{"windows": `%USERPROFILE%/.copilot/instructions.md`}, Strategy: "copy",
		}},
	}}}
	writeJSONFixture(t, filepath.Join(repo, "config", "apps.json"), manifest)
	writeFixture(t, filepath.Join(repo, "skills", "common", "example", "SKILL.md"), "skill")
	writeFixture(t, filepath.Join(repo, "behavior", "copilot.md"), "instructions")
	if err := os.MkdirAll(filepath.Join(home, ".copilot"), 0o755); err != nil {
		t.Fatal(err)
	}

	if err := SyncSkillsHub(repo, home, "windows"); err != nil {
		t.Fatal(err)
	}
	assertFileContent(t, filepath.Join(home, ".copilot", "skills", "example", "SKILL.md"), "skill")
	assertFileContent(t, filepath.Join(home, ".copilot", "instructions.md"), "instructions")
}

func TestMergeJSONFilesPreservesUnmanagedKeys(t *testing.T) {
	root := t.TempDir()
	source, target := filepath.Join(root, "managed.json"), filepath.Join(root, "target.json")
	writeFixture(t, source, `{"skills":{"paths":["__COPILOT_SKILLS_DIR__"]},"model":"managed"}`)
	writeFixture(t, target, `{"theme":"dark","model":"old"}`)

	if err := mergeJSONFiles(source, target, `C:\Users\Test`); err != nil {
		t.Fatal(err)
	}
	var result map[string]any
	data, _ := os.ReadFile(target)
	if err := json.Unmarshal(data, &result); err != nil {
		t.Fatal(err)
	}
	if result["theme"] != "dark" || result["model"] != "managed" {
		t.Fatalf("unexpected merge: %#v", result)
	}
}

func writeFixture(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func writeJSONFixture(t *testing.T, path string, value any) {
	t.Helper()
	data, err := json.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}
	writeFixture(t, path, string(data))
}

func assertFileContent(t *testing.T, path, want string) {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != want {
		t.Fatalf("%s = %q, want %q", path, data, want)
	}
}
