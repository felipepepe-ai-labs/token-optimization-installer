package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/huh"
	"github.com/charmbracelet/x/term"
	"github.com/token-optimizer/installer/internal/install"
)

const (
	componentCodebaseMemory = "codebase-memory-mcp"
	componentCodebaseUI     = "codebase-memory-ui"
	componentRTK            = "rtk"
	componentHeadroom       = "headroom"
	componentEngram         = "engram"
	componentVSCode         = "vscode"
	componentSkillsHub      = "skills-hub"
	componentDocsAgent      = "docs-agent-vscode"
)

func main() {
	dryRun := flag.Bool("dry-run", false, "show actions without changing the system")
	workspace := flag.String("workspace", ".", "workspace where .vscode/mcp.json is configured")
	flag.Parse()

	clearScreen(os.Stdout)
	writeOwl(os.Stdout, terminalWidth())
	selected, err := selectComponents()
	if err != nil {
		if errors.Is(err, huh.ErrUserAborted) {
			fmt.Println("\nInstalación cancelada.")
			return
		}
		fmt.Fprintf(os.Stderr, "\nSelección cancelada: %v\n", err)
		os.Exit(1)
	}
	if len(selected) == 0 {
		fmt.Println("No se seleccionó ningún componente.")
		return
	}
	plan := planFromSelection(selected, *workspace)

	runner := install.Runner{DryRun: *dryRun, Out: os.Stdout}
	if err = install.Execute(plan, runner); err != nil {
		fmt.Fprintf(os.Stderr, "\nError: %v\n", err)
		os.Exit(1)
	}
	abs, _ := filepath.Abs(plan.Workspace)
	fmt.Printf("\nInstalación completada. Workspace: %s\n", abs)
}

func clearScreen(writer io.Writer) {
	fmt.Fprint(writer, "\x1b[2J\x1b[H")
}

func terminalWidth() int {
	width, _, err := term.GetSize(os.Stdout.Fd())
	if err != nil || width <= 0 {
		return 80
	}
	return width
}

func writeOwl(writer io.Writer, width int) {
	const artWidth = 10
	padding := 0
	if width > artWidth {
		padding = (width - artWidth) / 2
	}
	prefix := strings.Repeat(" ", padding)
	for _, line := range []string{"    ___", "   (o,o)", `  /)  )\`, `---"--"---`} {
		fmt.Fprintln(writer, prefix+line)
	}
	fmt.Fprintln(writer)
}

func selectComponents() ([]string, error) {
	statuses := install.DetectUpdates()
	selected := []string{componentVSCode}
	options := []huh.Option[string]{
		componentOption("codebase-memory-mcp", componentCodebaseMemory, statuses[componentCodebaseMemory], &selected),
		huh.NewOption("Codebase Memory Graph UI — http://localhost:9749", componentCodebaseUI),
		componentOption("RTK — Rust Token Killer", componentRTK, statuses[componentRTK], &selected),
		componentOption("Headroom — herramientas MCP", componentHeadroom, statuses[componentHeadroom], &selected),
		componentOption("Engram — memoria persistente MCP", componentEngram, statuses[componentEngram], &selected),
		huh.NewOption("VS Code — configurar .vscode/mcp.json", componentVSCode).Selected(true),
		componentOption("Skills Hub — instalación nativa Windows", componentSkillsHub, statuses[componentSkillsHub], &selected),
		componentOption("Docs Agent — extensión para VS Code", componentDocsAgent, statuses[componentDocsAgent], &selected),
	}
	field := huh.NewMultiSelect[string]().
		Title("Token Optimizer Installer — selecciona componentes").
		Description("↑/↓ navegar · espacio marcar · Enter instalar · Esc salir").
		Options(options...).
		Value(&selected)
	form := huh.NewForm(huh.NewGroup(field)).WithKeyMap(installerKeyMap())
	return selected, form.Run()
}

func componentOption(label, value string, status install.UpdateStatus, selected *[]string) huh.Option[string] {
	switch {
	case !status.Installed:
		label += " — no instalada"
	case status.UpdateAvailable:
		label += fmt.Sprintf(" — actualizar %s → %s", status.Current, status.Latest)
	case status.Checked:
		label += fmt.Sprintf(" — actualizada (%s)", status.Current)
	default:
		label += " — instalada; versión no comprobada"
	}
	option := huh.NewOption(label, value)
	if status.ShouldSelect() {
		*selected = append(*selected, value)
		option = option.Selected(true)
	}
	return option
}

func installerKeyMap() *huh.KeyMap {
	keyMap := huh.NewDefaultKeyMap()
	keyMap.Quit = key.NewBinding(
		key.WithKeys("esc", "ctrl+c"),
		key.WithHelp("esc", "salir"),
	)
	return keyMap
}

func planFromSelection(selected []string, workspace string) install.Plan {
	chosen := make(map[string]bool, len(selected))
	for _, component := range selected {
		chosen[component] = true
	}
	return install.Plan{
		CodebaseMemory: chosen[componentCodebaseMemory],
		CodebaseUI:     chosen[componentCodebaseUI],
		RTK:            chosen[componentRTK],
		Headroom:       chosen[componentHeadroom],
		Engram:         chosen[componentEngram],
		VSCode:         chosen[componentVSCode],
		SkillsHub:      chosen[componentSkillsHub],
		DocsAgent:      chosen[componentDocsAgent],
		Workspace:      workspace,
	}
}
