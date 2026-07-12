# Token Optimizer Installer

Interactive installer for Linux and Windows. It installs codebase-memory-mcp,
RTK (Rust Token Killer), Headroom MCP tools, Engram persistent memory, Skills Hub,
Docs Agent, and optionally configures VS Code.

The installer opens a checkbox list. Use `↑/↓` to navigate, `space` to toggle
components, `Enter` to install the selection, or `Esc` to exit without making
changes. Missing and outdated tools start selected; tools already at their
latest detected version stay unselected. If an online version check fails, an
existing tool is left unselected instead of being reinstalled speculatively.
The terminal is cleared immediately before displaying the checklist.

The optional Codebase Memory Graph UI checkbox starts the MCP server with its
web explorer enabled on `http://localhost:9749`. It is disabled by default and
does not create a background process during installation; the UI starts when
VS Code launches the configured MCP server.

Selecting RTK also installs its global Copilot/VS Code command-rewrite hooks
non-interactively (`rtk init -g --auto-patch`). RTK creates backups before
patching agent configuration, and the hooks can be removed with
`rtk init -g --uninstall`.

Skills Hub is installed natively by the Go binary on Windows. The installer
clones the stable `v2.8.0` catalog and processes its manifest directly, without
requiring Bash, rsync, Node.js, or pnpm. Git must be available in `PATH`.

Docs Agent is installed from the stable `v0.3.5` tag. Building its VSIX requires
Git, Node.js 22+, pnpm, the VS Code `code` CLI, and Windows C++ build tools for
its native SQLite dependency.

Engram is compiled from its official Go module into the installer's private
binary directory and registered in VS Code with the agent MCP tool profile.
It requires Go 1.24 or newer and does not depend on WSL on Windows.

Python tools are installed in a private virtual environment under the user's
configuration directory. The installer never uses `pip --user` or
`--break-system-packages`, so it is compatible with PEP 668 distributions.
Downloads and archive extraction use a private temporary directory on the
user's main disk instead of the system `/tmp`, which may be a small tmpfs.

Building from source requires Go 1.23 or newer; the module pins the Go 1.24.4
toolchain and all TUI dependencies in `go.mod` and `go.sum`.

```bash
go run .
go run . --dry-run
go build -o token-optimizer-installer .
```

Cross-compile release binaries:

```bash
GOOS=linux GOARCH=amd64 go build -o dist/token-optimizer-installer-linux-amd64 .
GOOS=windows GOARCH=amd64 go build -o dist/token-optimizer-installer-windows-amd64.exe .
```

VS Code configuration is merged into `<workspace>/.vscode/mcp.json`; existing
servers and inputs are preserved.
