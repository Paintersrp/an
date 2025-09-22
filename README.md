# Atomic Notes CLI

Atomic Notes CLI (`an`) is a terminal-first companion for maintaining an Obsidian-style vault of "atomic" knowledge notes. It wraps a collection of Go commands and Bubble Tea interfaces so you can create, triage, and revisit notes without leaving the keyboard while still launching into your configured editor when it's time to write.

## Prerequisites

Before you build or run the CLI make sure you have the following tooling installed:

- **Go 1.22+** – the module is authored for Go 1.22.2 and relies on modern toolchain behavior.
- **Git** – required if you plan to clone the repository instead of using `go install`.
- **Terminal editor** – Neovim is the default and currently the only editor accepted by the interactive initializer, but the runtime can launch Vim, Nano, VS Code, or Obsidian if you set the `editor` field manually.
- **Desktop opener** – the CLI shells out to `open`/`xdg-open`/`cmd /c start` when you request Obsidian or VS Code; ensure one of those launchers is present on your OS.

Optional integrations, such as clipboard-driven note creation (`an new --paste`), rely on your OS clipboard provider via [`github.com/atotto/clipboard`].

## Installation

You can install prebuilt binaries with `go install` or build locally:

```bash
# Install the latest tagged version into your GOPATH/bin
GO111MODULE=on go install github.com/Paintersrp/an@latest

# Or clone and build from source
git clone https://github.com/Paintersrp/an.git
cd an
go build ./...
```

Add the resulting binary directory to your `PATH` so the `an` command is available globally.

## Configuration & Vault Layout

The CLI keeps user preferences in `~/.an/cfg.yaml`. The `initialize` command will scaffold the file and prompt for your vault path and editor, then create an `atoms/` subdirectory where new notes live by default. If you prefer to edit the configuration manually, the following fields are recognized:

```yaml
vaultdir: /absolute/path/to/your/vault
editor: nvim            # Required; must be one of the supported editors
nvimargs: ''            # Optional flags passed to Neovim
fsmode: confirm         # strict | confirm | free controls subdirectory enforcement
pinned_file: ''         # Absolute path of the currently pinned note
pinned_task_file: ''    # Absolute path of the currently pinned task note
subdirs:
  - atoms               # Additional writing destinations inside the vault
named_pins: {}
named_task_pins: {}
```

During note creation you can target other subdirectories with `--subdir`, but ensure they exist (or add them with `an add-subdir`) when running in `strict` or `confirm` filesystem modes. Archival workflows expect `archive/` and `trash/` folders alongside your writing area so the TUI views and file handlers can move notes without prompting.

## Quick Start

```bash
# 1. Initialize configuration and create your vault directories
an init

# 2. Create a note using the default "atoms" subdirectory
an new "first atomic note" "zettelkasten cli" --pin

# 3. Launch the interactive TUI to browse and manage notes
an notes --view default
```

When the Bubble Tea interface opens you will see a scrollable list of notes, a preview pane, and an on-demand help panel. Use <kbd>↑</kbd>/<kbd>↓</kbd> or <kbd>j</kbd>/<kbd>k</kbd> to move, <kbd>enter</kbd> to open the highlighted note in your editor, <kbd>tab</kbd> to toggle focus between the list and detail panel, and <kbd>?</kbd>/<kbd>h</kbd> to expand the full key binding cheat sheet. Common actions include <kbd>c</kbd> to create a note, <kbd>r</kbd> to rename, <kbd>y</kbd> to copy, <kbd>v</kbd> to switch views, and number keys <kbd>1</kbd>–<kbd>5</kbd> to jump between default, orphan, unfulfilled, archive, and trash views respectively.

Run `an --help` or any subcommand with `--help` to explore the rest of the command surface (journal, todo, settings, pin management, symlinks, etc.).

## Roadmap & Further Reading

For upcoming features, architecture notes, and broader project context, read the [Planning roadmap](Planning%20v3.md) document in the repository root.
