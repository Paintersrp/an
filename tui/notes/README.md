# Notes Module

## Overview
The Notes module is designed to manage and interact with note files within a terminal user interface (TUI). It provides a range of features to view, edit, and organize notes efficiently.

## Main Files

### `notes.go`
The central file of the Notes module, responsible for the overall management of the TUI, including initialization, user input handling, and interface rendering.

#### Purpose
`notes.go` is the backbone of the Notes module. It orchestrates the interactions between the user and the application, processes commands, and renders the TUI. It's where the main logic resides, making it essential for the module's functionality.

#### File Dependencies
- **Module Files**:
  - `delegate.go`: Defines key bindings and their associated actions.
  - `items.go` (if applicable): Manages the list data structure and its operations.
  - `keys.go`
  - `modes.go`
  - `styles.go`
  - `utils.go`

- **Internal Modules**:
  - `fs/zet`: Deals with file system operations specific to Zettelkasten note-taking method.
  - `utils`: Provides some global utility functions
  - `config`: Handles configuration settings.
  - `cache`: Used to manage caching of note previews.

- **External Modules**:
  - `charmbracelet` packages: Used for building the TUI components.

### `delegate.go`
Used by `notes.go` to set up the delegate keys, which define the actions triggered by key presses in the TUI.

#### Purpose
The `delegate.go` file separates the concerns of key binding definitions and actions from the main `notes.go` file. This allows for a cleaner codebase and easier addition of new functionalities.

#### File Dependencies
- **External Packages**:
  - `config`: Accesses configuration settings.
  - `charmbracelet/bubbles/key`: Used for defining key bindings.
  - `charmbracelet/bubbles/list`: Utilized for list operations in the TUI.
  - `charmbracelet/bubbletea`: The TUI framework used throughout the module.

### `items.go`
Handles the creation and management of list items within the TUI.

#### Purpose
`items.go` defines the `ListItem` struct and associated methods that determine how each item is displayed, described, and filtered within the TUI. It plays a crucial role in how information is presented to the user.

### `keys.go`
Defines the key bindings for the TUI's list model and provides help text for the menus.

#### Purpose
`keys.go` sets up the key bindings for the `notes.go` model list, enabling user interaction with the TUI through keyboard shortcuts. It also defines the help text displayed in the short and long help menus, aiding users in understanding the available commands.

#### File Dependencies
- **External Packages**:
  - `charmbracelet/bubbles/key`: Used for defining key bindings.

### `modes.go`
Defines different modes for listing notes and their configurations.

#### Purpose
`modes.go` contains the `ModeConfig` struct and functions to generate and manage different modes for displaying notes, such as default, archive, and orphan modes. It allows for dynamic filtering of notes based on the selected mode.

#### File Dependencies
- **Internal Modules**:
  - `fs/fzf`: Provides file listing capabilities with exclusion filters.

### `styles.go`
Contains style definitions for the TUI components.

#### Purpose
`styles.go` defines various styles using the `lipgloss` package to enhance the visual appearance of the TUI. It includes styles for the application layout, titles, input fields, status messages, and list items.

#### File Dependencies
- **External Modules**:
  - `charmbracelet/lipgloss`: Used for styling the TUI components.


### `utils.go`
Provides utility functions for file and directory operations within the TUI.

#### Purpose
`utils.go` serves as a utility belt for the Notes module. It includes functions for parsing note files, extracting metadata from notes, and managing subdirectories. This file is crucial for handling the data processing aspect of the TUI, ensuring that the notes are displayed with accurate and up-to-date information.

#### File Dependencies
- **External Modules**:
  - `list`: From `github.com/charmbracelet/bubbles/list`, used to manage list items in the TUI.
  - `yaml`: From `gopkg.in/yaml.v2`, utilized for parsing YAML content.

---
