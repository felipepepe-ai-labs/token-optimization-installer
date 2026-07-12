package install

import (
	"context"
	"encoding/json"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"sync"
	"time"
)

type UpdateStatus struct {
	Installed       bool
	Checked         bool
	UpdateAvailable bool
	Current         string
	Latest          string
}

func (status UpdateStatus) ShouldSelect() bool {
	return !status.Installed || status.UpdateAvailable
}

var versionPattern = regexp.MustCompile(`(?i)\bv?(\d+\.\d+\.\d+(?:[-+][0-9a-z.-]+)?)\b`)

func DetectUpdates() map[string]UpdateStatus {
	checks := map[string]func() UpdateStatus{
		"codebase-memory-mcp": func() UpdateStatus {
			return checkRemoteTool(privatePythonTool("codebase-memory-mcp"), "codebase-memory-mcp", "--version", "https://pypi.org/pypi/codebase-memory-mcp/json", true)
		},
		"rtk": func() UpdateStatus {
			return checkRemoteTool("", "rtk", "--version", "https://api.github.com/repos/rtk-ai/rtk/releases/latest", false)
		},
		"headroom": func() UpdateStatus {
			return checkRemoteTool(privatePythonTool("headroom"), "headroom", "--version", "https://pypi.org/pypi/headroom-ai/json", true)
		},
		"engram": func() UpdateStatus {
			return checkRemoteTool(privateBinary("engram"), "engram", "version", "https://api.github.com/repos/Gentleman-Programming/engram/releases/latest", false)
		},
		"skills-hub": func() UpdateStatus {
			return checkPinnedCheckout(filepath.Join(userConfigDir(), "skills-hub"), skillsHubVersion)
		},
		"docs-agent-vscode": func() UpdateStatus {
			return checkPinnedCheckout(filepath.Join(userConfigDir(), "token-optimizer", "docs-agent-vscode"), docsAgentVersion)
		},
	}
	statuses := make(map[string]UpdateStatus, len(checks))
	var mutex sync.Mutex
	var group sync.WaitGroup
	for name, check := range checks {
		group.Add(1)
		go func() {
			defer group.Done()
			status := check()
			mutex.Lock()
			statuses[name] = status
			mutex.Unlock()
		}()
	}
	group.Wait()
	return statuses
}

func checkRemoteTool(preferred, fallback, versionArg, endpoint string, pypi bool) UpdateStatus {
	command := preferred
	if command == "" || !fileExists(command) {
		var err error
		command, err = exec.LookPath(fallback)
		if err != nil {
			return UpdateStatus{}
		}
	}
	ctx, cancel := context.WithTimeout(context.Background(), 4*time.Second)
	defer cancel()
	output, err := exec.CommandContext(ctx, command, versionArg).CombinedOutput()
	if err != nil {
		return UpdateStatus{Installed: true}
	}
	current := extractVersion(string(output))
	status := UpdateStatus{Installed: true, Current: current}
	latest, err := fetchLatestVersion(endpoint, pypi)
	if err != nil || current == "" || latest == "" {
		return status
	}
	status.Checked = true
	status.Latest = latest
	status.UpdateAvailable = compareVersions(current, latest) < 0
	return status
}

func fetchLatestVersion(endpoint string, pypi bool) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 4*time.Second)
	defer cancel()
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return "", err
	}
	request.Header.Set("User-Agent", "token-optimizer-installer")
	response, err := http.DefaultClient.Do(request)
	if err != nil {
		return "", err
	}
	defer response.Body.Close()
	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		return "", &httpError{status: response.Status}
	}
	var payload struct {
		TagName string `json:"tag_name"`
		Info    struct {
			Version string `json:"version"`
		} `json:"info"`
	}
	if err := json.NewDecoder(response.Body).Decode(&payload); err != nil {
		return "", err
	}
	if pypi {
		return strings.TrimPrefix(payload.Info.Version, "v"), nil
	}
	return strings.TrimPrefix(payload.TagName, "v"), nil
}

func checkPinnedCheckout(directory, target string) UpdateStatus {
	if !fileExists(filepath.Join(directory, ".git")) {
		return UpdateStatus{Latest: strings.TrimPrefix(target, "v")}
	}
	ctx, cancel := context.WithTimeout(context.Background(), 4*time.Second)
	defer cancel()
	output, err := exec.CommandContext(ctx, "git", "-C", directory, "describe", "--tags", "--exact-match").CombinedOutput()
	current := strings.TrimPrefix(strings.TrimSpace(string(output)), "v")
	latest := strings.TrimPrefix(target, "v")
	return UpdateStatus{
		Installed:       true,
		Checked:         err == nil,
		UpdateAvailable: err != nil || compareVersions(current, latest) < 0,
		Current:         current,
		Latest:          latest,
	}
}

type httpError struct{ status string }

func (err *httpError) Error() string { return err.status }

func privatePythonTool(name string) string {
	bin := "bin"
	if runtime.GOOS == "windows" {
		bin, name = "Scripts", name+".exe"
	}
	return filepath.Join(userConfigDir(), "token-optimizer", "python", bin, name)
}

func privateBinary(name string) string {
	if runtime.GOOS == "windows" {
		name += ".exe"
	}
	return filepath.Join(userConfigDir(), "token-optimizer", "bin", name)
}

func userConfigDir() string {
	directory, _ := os.UserConfigDir()
	return directory
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func extractVersion(value string) string {
	match := versionPattern.FindStringSubmatch(value)
	if len(match) < 2 {
		return ""
	}
	return match[1]
}

func compareVersions(left, right string) int {
	left = strings.TrimPrefix(strings.TrimSpace(left), "v")
	right = strings.TrimPrefix(strings.TrimSpace(right), "v")
	leftParts := strings.SplitN(left, "-", 2)
	rightParts := strings.SplitN(right, "-", 2)
	for index := 0; index < 3; index++ {
		leftValue, rightValue := 0, 0
		if index < len(strings.Split(leftParts[0], ".")) {
			leftValue = parseVersionNumber(strings.Split(leftParts[0], ".")[index])
		}
		if index < len(strings.Split(rightParts[0], ".")) {
			rightValue = parseVersionNumber(strings.Split(rightParts[0], ".")[index])
		}
		if leftValue < rightValue {
			return -1
		}
		if leftValue > rightValue {
			return 1
		}
	}
	return 0
}

func parseVersionNumber(value string) int {
	number := 0
	for _, character := range value {
		if character < '0' || character > '9' {
			break
		}
		number = number*10 + int(character-'0')
	}
	return number
}
