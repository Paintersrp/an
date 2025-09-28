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
editor_template:
  exec: ''              # Optional wrapper command (supports {cmd}, {args}, {file}, {vault}, {relative}, {filename})
  args: []
  wait: null            # Override whether the CLI should wait for the command to exit
  silence: null         # Silence stdout/stderr when launching asynchronous editors
hooks:
  pre_open: []          # Commands run before opening a note (placeholders match the editor template)
  post_open: []         # Commands run after the editor exits
  post_create: []       # Commands run after a note is written to disk
subdirs:
  - atoms               # Additional writing destinations inside the vault
named_pins: {}
named_task_pins: {}
```

The `editor_template` block lets you wrap the built-in editor command with a custom launcher (for example, opening Neovim in a
dedicated terminal tab or forwarding over SSH). Use `{cmd}` to reference the resolved editor binary and `{args}` to inject the
default argument list, or substitute `{file}`, `{vault}`, `{relative}`, and `{filename}` directly. Automation hooks follow the
same placeholder rules so you can trigger local sync jobs or backups before and after editing and immediately after creating a
new note.

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

When the Bubble Tea interface opens you will see a scrollable list of notes, a preview pane, and an on-demand help panel. Use <kbd>↑</kbd>/<kbd>↓</kbd> or <kbd>j</kbd>/<kbd>k</kbd> to move, <kbd>enter</kbd> to open the highlighted note in your editor, <kbd>tab</kbd> to toggle focus between the list and detail panel, and <kbd>?</kbd>/<kbd>h</kbd> to expand the full key binding cheat sheet. Common actions include <kbd>c</kbd> to create a note, <kbd>r</kbd> to rename, <kbd>y</kbd> to copy, <kbd>v</kbd> to switch views, and number keys <kbd>1</kbd>–<kbd>5</kbd> to jump between default, orphan, unfulfilled, archive, and trash views respectively. Switch between the notes, tasks, and journal workspaces with <kbd>n</kbd>, <kbd>i</kbd>, and <kbd>l</kbd>. If you maintain multiple vaults, press <kbd>ctrl</kbd>+<kbd>w</kbd> to cycle through the configured workspaces; the active workspace is displayed alongside the view picker.

### Inline editing & captures

Press <kbd>e</kbd> to open the highlighted note inside an inline editor without leaving the TUI. The textarea honours <kbd>ctrl+s</kbd> for save, <kbd>ctrl+r</kbd> to reload the on-disk version, and <kbd>esc</kbd> to discard changes (press twice to confirm if the buffer is dirty). External modifications are detected—when the backing file changes the editor warns and requires a second save to overwrite so you can reload the newer content instead.

For quick captures, hit <kbd>q</kbd> to spawn a scratch buffer. Saving with <kbd>ctrl+s</kbd> writes the content into the first configured subdirectory (or the vault root when none is set) using a timestamped filename and refreshes the list so the new note is immediately available.

Run `an --help` or any subcommand with `--help` to explore the rest of the command surface (journal, settings, pin management, symlinks, etc.).

## Smarter templates & guided capture

Templates can now declare their own metadata requirements and helper text. Each `.tmpl` file may begin with an embedded manifest
comment (see `internal/templater/templates/project.tmpl` for a complete example) describing extra fields, select options, and
default values. When you create a note with one of these templates the CLI renders an interactive prompt before opening your
editor so the answers are recorded in the Markdown front matter automatically—no more ad-hoc status keys.

To make the flow easier to adopt, a dedicated `capture` command guides you through template selection, previews, upstream
assignment, and view targeting:

```bash
an capture --template project-release
```

The capture wizard works with workspace-defined views and any template-specific metadata. You can skip the preview step with
`--no-preview` or pre-fill values such as `--title` and `--view` when scripting.

Capture rules defined in your workspace configuration can also pre-populate tags and front matter before the note hits disk.
Rules match on template names and optional upstream prefixes, making it easy to apply consistent metadata for release notes,
meeting logs, or any other templated workflow. If a rule marks a clipboard requirement it only fires when a non-empty clipboard
value is present, which is perfect for "paste into note" shortcuts.

Any tags you enter manually are deduplicated alongside the rule-provided set so you can confidently re-type common keywords
without worrying about duplicates in front matter. When you just want to inspect the metadata overlay, run `an capture --dry-run`
to see the merged tags and front matter preview without creating a file. See [Capture rules & automation](docs/capture-rules.md)
for configuration examples.

## Testing

Run the unit suite before sending a pull request to confirm core flows still pass:

```bash
go test ./...
```

This exercises the configuration loader, view selection helpers, the Markdown parser, and the templating system so regressions in those critical packages are caught early.

## Roadmap & Further Reading

For upcoming features, architecture notes, and broader project context, read the [Planning roadmap](Planning%20v3.md) document in the repository root.
