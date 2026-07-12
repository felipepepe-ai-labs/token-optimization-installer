package main

import (
	"bytes"
	"testing"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
)

func TestClearScreen(t *testing.T) {
	var output bytes.Buffer
	clearScreen(&output)
	if output.String() != "\x1b[2J\x1b[H" {
		t.Fatalf("unexpected clear sequence: %q", output.String())
	}
}

func TestWriteOwl(t *testing.T) {
	var output bytes.Buffer
	writeOwl(&output, 20)
	want := "         ___\n        (o,o)\n       /)  )\\\n     ---\"--\"---\n\n"
	if output.String() != want {
		t.Fatalf("unexpected owl:\n%s", output.String())
	}
}

func TestPlanFromSelection(t *testing.T) {
	plan := planFromSelection([]string{componentRTK, componentVSCode}, "project")

	if plan.CodebaseMemory || !plan.RTK || plan.Headroom || !plan.VSCode {
		t.Fatalf("unexpected plan: %+v", plan)
	}
	if plan.Workspace != "project" {
		t.Fatalf("unexpected workspace: %q", plan.Workspace)
	}
}

func TestPlanAllowsCodebaseUIWithoutStandardCheckbox(t *testing.T) {
	plan := planFromSelection([]string{componentCodebaseUI, componentVSCode}, "project")
	if plan.CodebaseMemory || !plan.CodebaseUI || !plan.VSCode {
		t.Fatalf("unexpected UI plan: %+v", plan)
	}
}

func TestInstallerKeyMapAllowsEscapeAndCtrlC(t *testing.T) {
	keyMap := installerKeyMap()
	for name, message := range map[string]tea.KeyMsg{
		"escape": {Type: tea.KeyEsc},
		"ctrl+c": {Type: tea.KeyCtrlC},
	} {
		t.Run(name, func(t *testing.T) {
			if !key.Matches(message, keyMap.Quit) {
				t.Fatalf("%s should cancel the installer", name)
			}
		})
	}
}
