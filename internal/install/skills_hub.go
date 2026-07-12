package install

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

const (
	skillsHubRepository = "https://github.com/FelipePepe/skills-hub.git"
	skillsHubVersion    = "v2.8.0"
)

type skillsHubManifest struct {
	Apps []skillsHubApp `json:"apps"`
}

type skillsHubApp struct {
	ID               string              `json:"id"`
	InstallPath      map[string]string   `json:"installPath"`
	AgentInstallPath map[string]string   `json:"agentInstallPath"`
	DetectPaths      map[string][]string `json:"detectPaths"`
	Sources          []string            `json:"sources"`
	AgentSources     []string            `json:"agentSources"`
	ConfigFiles      []skillsHubConfig   `json:"configFiles"`
}

type skillsHubConfig struct {
	Source   string            `json:"source"`
	Target   map[string]string `json:"target"`
	Strategy string            `json:"strategy"`
	BlockID  string            `json:"blockId"`
}

func InstallSkillsHub(runner Runner) error {
	configDir, err := os.UserConfigDir()
	if err != nil {
		return fmt.Errorf("resolve user config directory: %w", err)
	}
	repoDir := filepath.Join(configDir, "skills-hub")
	if _, err := os.Stat(filepath.Join(repoDir, ".git")); os.IsNotExist(err) {
		if err := runner.Run("git", "clone", "--branch", skillsHubVersion, "--depth", "1", skillsHubRepository, repoDir); err != nil {
			return err
		}
	} else {
		if err := runner.Run("git", "-C", repoDir, "fetch", "--depth", "1", "origin", "tag", skillsHubVersion); err != nil {
			return err
		}
		if err := runner.Run("git", "-C", repoDir, "checkout", "--detach", skillsHubVersion); err != nil {
			return fmt.Errorf("skills-hub checkout has local changes or cannot select %s: %w", skillsHubVersion, err)
		}
	}
	if runner.DryRun {
		fmt.Fprintf(runner.Out, "→ sync Skills Hub %s natively\n", repoDir)
		return nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}
	return SyncSkillsHub(repoDir, home, runtime.GOOS)
}

func SyncSkillsHub(repoDir, home, goos string) error {
	platform := "linux"
	if goos == "windows" {
		platform = "windows"
	}
	data, err := os.ReadFile(filepath.Join(repoDir, "config", "apps.json"))
	if err != nil {
		return err
	}
	var manifest skillsHubManifest
	if err := json.Unmarshal(data, &manifest); err != nil {
		return fmt.Errorf("parse skills-hub manifest: %w", err)
	}
	for _, app := range manifest.Apps {
		detected := false
		for _, raw := range app.DetectPaths[platform] {
			if _, err := os.Stat(expandUserPath(raw, home)); err == nil {
				detected = true
				break
			}
		}
		if !detected {
			continue
		}
		if target := expandUserPath(app.InstallPath[platform], home); target != "" {
			if err := requireInsideHome(target, home); err != nil {
				return err
			}
			for _, source := range app.Sources {
				if err := copyChildren(filepath.Join(repoDir, filepath.FromSlash(source)), target, true); err != nil {
					return fmt.Errorf("sync %s skills: %w", app.ID, err)
				}
			}
		}
		if target := expandUserPath(app.AgentInstallPath[platform], home); target != "" {
			if err := requireInsideHome(target, home); err != nil {
				return err
			}
			for _, source := range app.AgentSources {
				if err := copyChildren(filepath.Join(repoDir, filepath.FromSlash(source)), target, false); err != nil {
					return fmt.Errorf("sync %s agents: %w", app.ID, err)
				}
			}
		}
		for _, item := range app.ConfigFiles {
			if err := installSkillsHubConfig(repoDir, home, platform, item); err != nil {
				return fmt.Errorf("configure %s: %w", app.ID, err)
			}
		}
	}
	return nil
}

func expandUserPath(raw, home string) string {
	if raw == "" {
		return ""
	}
	value := strings.ReplaceAll(raw, "%USERPROFILE%", home)
	if value == "~" {
		return home
	}
	if strings.HasPrefix(value, "~/") || strings.HasPrefix(value, `~\`) {
		return filepath.Join(home, value[2:])
	}
	return filepath.Clean(value)
}

func requireInsideHome(target, home string) error {
	relative, err := filepath.Rel(filepath.Clean(home), filepath.Clean(target))
	if err != nil || relative == ".." || strings.HasPrefix(relative, ".."+string(filepath.Separator)) || filepath.IsAbs(relative) {
		return fmt.Errorf("skills-hub target escapes user profile: %s", target)
	}
	return nil
}

func copyChildren(source, target string, directories bool) error {
	entries, err := os.ReadDir(source)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(target, 0o755); err != nil {
		return err
	}
	for _, entry := range entries {
		if strings.HasPrefix(entry.Name(), ".") || strings.HasPrefix(entry.Name(), "_") || entry.IsDir() != directories {
			continue
		}
		destination := filepath.Join(target, entry.Name())
		if directories {
			if err := os.RemoveAll(destination); err != nil {
				return err
			}
		}
		if err := copyTree(filepath.Join(source, entry.Name()), destination); err != nil {
			return err
		}
	}
	return nil
}

func copyTree(source, target string) error {
	info, err := os.Stat(source)
	if err != nil {
		return err
	}
	if !info.IsDir() {
		data, err := os.ReadFile(source)
		if err != nil {
			return err
		}
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			return err
		}
		return os.WriteFile(target, data, info.Mode().Perm())
	}
	return filepath.WalkDir(source, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		relative, err := filepath.Rel(source, path)
		if err != nil {
			return err
		}
		destination := filepath.Join(target, relative)
		if entry.IsDir() {
			return os.MkdirAll(destination, 0o755)
		}
		return copyTree(path, destination)
	})
}

func installSkillsHubConfig(repoDir, home, platform string, item skillsHubConfig) error {
	source := filepath.Join(repoDir, filepath.FromSlash(item.Source))
	target := expandUserPath(item.Target[platform], home)
	if target == "" {
		return nil
	}
	if err := requireInsideHome(target, home); err != nil {
		return err
	}
	switch item.Strategy {
	case "copy":
		return copyTree(source, target)
	case "json-merge":
		return mergeJSONFiles(source, target, home)
	case "markdown-managed-block":
		return mergeMarkdownBlock(source, target, item.BlockID)
	default:
		return fmt.Errorf("unsupported strategy %q", item.Strategy)
	}
}

func mergeJSONFiles(source, target, home string) error {
	sourceData, err := os.ReadFile(source)
	if err != nil {
		return err
	}
	skillsPath := strings.ReplaceAll(filepath.Join(home, ".copilot", "skills"), `\`, "/")
	sourceData = []byte(strings.ReplaceAll(string(sourceData), "__COPILOT_SKILLS_DIR__", skillsPath))
	managed, current := map[string]any{}, map[string]any{}
	if err := json.Unmarshal(sourceData, &managed); err != nil {
		return err
	}
	if data, err := os.ReadFile(target); err == nil {
		if err := json.Unmarshal(data, &current); err != nil {
			return err
		}
	}
	mergeMaps(current, managed)
	data, err := json.MarshalIndent(current, "", "  ")
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		return err
	}
	return os.WriteFile(target, append(data, '\n'), 0o644)
}

func mergeMaps(target, managed map[string]any) {
	for key, value := range managed {
		if child, ok := value.(map[string]any); ok {
			existing, _ := target[key].(map[string]any)
			if existing == nil {
				existing = map[string]any{}
			}
			mergeMaps(existing, child)
			target[key] = existing
		} else {
			target[key] = value
		}
	}
}

func mergeMarkdownBlock(source, target, blockID string) error {
	managed, err := os.ReadFile(source)
	if err != nil {
		return err
	}
	current, _ := os.ReadFile(target)
	start := "<!-- skills-hub:" + blockID + ":start -->"
	end := "<!-- skills-hub:" + blockID + ":end -->"
	text := string(current)
	if from := strings.Index(text, start); from >= 0 {
		if to := strings.Index(text[from:], end); to >= 0 {
			text = text[:from] + text[from+to+len(end):]
		}
	}
	text = strings.TrimSpace(text) + "\n\n" + start + "\n" + strings.TrimSpace(string(managed)) + "\n" + end + "\n"
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		return err
	}
	return os.WriteFile(target, []byte(text), 0o644)
}
