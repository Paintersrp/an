---
title: Atomic Notes Roadmap v1
up: "[[atomic-notes]]"
date: 20240501082615
tags:
  - atomic-notes
  - roadmaps
  - features
---
## 🎯 Version 1.0 Objectives

---
### New Features

- [[Templates]] allow creating notes quickly into various formats.
- [[Config]] global configuration to ensure systems work currently and consistently.
- [[Zettelkasten Note]] aids in organizing and structuring the handling of all aspects of notes.
- [[Fuzzyfinder]] allows efficient note searching and filtering for various systems.
- [[Parser]] parses and analyzes note metadata and frontmatter data.
- [[TUI]] sets up interactive displays for various commands.
    - [[TUI List]] creates list view of config with input views to make edits.
    - [[TUI Initialize]] sets up the initialize command interface for walking through setting up.

### New Commands

- [[New Command]] aids in creating notes using default templates or user defined templates. Allows parameters for passing a title, tags, links, an upstream file, and scope.
- [[Day Command]] sets up a structured daily note for starting the day, reflecting on during the day, and finalizing at the end of day. Leads into the next day.
- [[Open Command]] allow fuzzy-finding notes in vault by title or tag.
- [[Echo Command]] allows echoing a quick message into a pinned file. Companion command is pin.
- [[Pin Command]] pins a file for quick echoing directly from the terminal.
- [[Open Pin Command]] opens the pinned file directly to editor.
- [[Vault Command]] opens the vault directory directly to editor.
- [[Init Command]] initializes config for CLI and walks through with TUI / reasonable defaults.
- [[Settings Command]] opens TUI for interactively adjusting config.
- [[Add Subdirectory Command]] adds allowed subdirectories to the config, required if in strict mode. Confirm or free mode do not need strict subdirectory management.


## 🗂️ Task Breakdown

---

### [[Templates]]

- [x] Load Templates from Directories
- [x] Pass from the Root On-wards
- [x] Internal Templates
    - [x] Atom / Zettel
    - [x] Project Overview (Map of Content)
    - [x] Project Roadmap (Version Planning)
    - [x] Project Stack Item (Stack Notes)
    - [x] Project Feature (Permanent Notes from Atoms)
    - [x] Day
- [x] User Defined Templates
    - [x] Parsing
    - [x] Validating/Available in Error Messages

### [[Config]]

- [x] Save to File
- [x] Open from Path
- [x] Add Available Sub-directories
- [x] Change Various Config Settings
- [x] Get Used Config Path
- [x] Static Ensure Exists
- [x] Static Get Config Path

### [[Zettelkasten Note]]

- [x] Get Filepath (using struct properties)
- [x] Ensure Path for Creation
- [x] File Exists Method
- [x] Create File Method
- [x] Integrate Templater
- [x] Template Parameters
- [x] Open Note Method
- [x] Static OpenFromPath Method
- [x] Static GetNotesInDirectory Method
- [x] Static Note Launch
- [x] Editor Opening Handling
    - [x] Neovim
    - [x] Obsidian
- [x] OS Handling
    - [x] Linux

### [[Parser]]

- [x] Walk Vault
- [x] [[Tag Handler]]
    - [x] Parse Tags
    - [x] Print Tag Counts
        - [x] Sortable
            - [x] Ascending
            - [x] Descending
    - [x] Show Tags Table
        - [x] Integrate [[TUI Table]] 
- [x] [[Task Handler]]
    - [x] Parse Tasks
    - [x] Print Tasks
    - [x] Add Task
    - [x] Sortable
        - [x] ID
            - [x] Ascending
            - [x] Descending
        - [x] Status
            - [x] Ascending
            - [x] Descending
    - [x] Show Tasks Table
        - [x] Integrate [[TUI Table]]

### [[Fuzzyfinder]]

- [x] Run 
    - [x] With Query
    - [x] With Return
    - [x] With Execute (Open)
- [x] Walk to List Files
- [x] Configurable Header
- [x] Markdown Preview
    - [x] Integrate [[Glamour]]
- [x] Add Tags to Title for Fuzzyfinding by Tag
- [x] Handle Errors

### [[TUI List]]

- [x] Help Keys
    - [x] Title Toggle
    - [x] Status Toggle
    - [x] Pagination Toggle
    - [x] Help Toggle
- [x] Filter Search Items
- [x] Edit Item
    - [x] Edit Toggle
    - [x] Edit Input View
      - [x] Enter Keypress (Save)
      - [x] Esc To Exit
    - [x] Save to File
- [x] Basic Styling

### [[TUI Initialize]]

- [x] Code Organization
- [x] Multi Input View
- [x] Default Placeholders
- [x] Escape Keys
- [x] Focus Keys
- [x] Tab Index
- [x] Submit Button
- [x] Setup Defaults and Generate File
- [x] Basic Styling

### [[Settings Command]]

- [ ] Command Tertiary Properties
- [ ] Initial TUI with Interactivity
    - [x] Common Key-press Handling for Exiting / Interactivity
    - [x] List Display with Input Change View
    - [ ] Organize Code
- [x] Change, commit, and save.
- [x] Validate Input

### [[Init Command]]

- [ ] Command Tertiary Properties
- [x] Initial TUI with Interactivity
- [x] Easy Defaults

### [[New Command]]

- [ ] Command Tertiary Properties
- [x] Template Integration
- [x] ZettelkastenNote Integration
- [x] Scoped (targeted to sub-directory) Creation
- [x] Title/Tags Parameters
- [x] Links/Upstream/Pin Flags
- [x] Conflict Handling

### [[Day Command]]

- [ ] Command Tertiary Properties
- [x] ZettelkastenNote Integration
- [x] Tags Parameter
- [x] Index/Links Flags

### [[Echo Command]]

- [ ] Command Tertiary Properties
- [ ] Validate Input
- [x] Utilize and Verify Pin
- [x] Write to File
- [x] Task Variant
    - [x] Priority Flag (-p)

### [[Pin Command]]

- [ ] Command Tertiary Properties
- [ ] Add Pin Check
- [x] Validate Input
- [x] Fuzzyfinding File Selection
- [x] Path Flag (Manual Pin)
- [x] Task Variant

### [[Open Command]]

- [ ] Command Tertiary Properties
- [x] Vault Flag 
- [x] Query Parameter

### [[Open Pin Command]]

- [ ] Command Tertiary Properties
- [x] Config Pinned File Validation

### [[Vault Command]]

- [ ] Command Tertiary Properties
- [x] Config Vault Directory Validation

### [[Add Subdirectory Command]]

- [ ] Command Tertiary Properties
- [x] Conflict Handling
- [ ] Input Validation

---

## 📌 Milestones

- Milestone 1: Initial Release
  - **Expected Date**: 2024-05-02
  - **Deliverables**:
    - Complete implementation of [[Templates]], [[Config]], and [[Zettelkasten Note Struct]].
    - Fully functional [[Fuzzyfinder]] and [[Parser]].
    - Launch of [[TUI]] with [[TUI List]] and [[TUI Initialize]] interfaces.
    - Release of [[New Command]], [[Day Command]], and [[Open Command]].
    - Integration of all remaining commands including [[Echo Command]], [[Pin Command]], and [[Vault Command]].
    - Completion of [[Settings Command]] and [[Init Command]] with TUI interactivity.
    - Finalization of [[Add Subdirectory Command]] and input validation.
    - Polishing and bug fixes.

- Milestone 2: Public Launch
  - **Expected Date**: 2024-06-01
  - **Deliverables**:
    - Public release of Atomic Notes v1.0.
    - Comprehensive documentation and user guides.
    - Community engagement and feedback incorporation.
    - Marketing and promotional activities.

---

## 📝 Additional Notes

This roadmap is subject to change based on project requirements and stakeholder feedback. Regular updates will be provided in the project repository. It's important to review and adjust the milestones as the project progresses to reflect any changes in scope or priorities.

- [[atomic-notes]]
