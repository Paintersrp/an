# Expansion Roadmap Ideas

This document captures medium-term feature expansions for the Atomic Notes CLI and converts them into self-contained tasks. The aim is to clarify why each direction matters, outline key technical considerations, and set up actionable TODO items to pursue.

## 1. Knowledge graph & backlink intelligence

**Vision.** Give writers a quick sense of how ideas interconnect by surfacing backlinks, forward links, and related context directly inside the TUI. This will make the tool more competitive with Obsidian's graph view without leaving the terminal.

**Why now.** The existing search index (`internal/search`) already parses links and front matter, so we have structured data to build on. Adding graph awareness would encourage better linking habits and faster discovery of forgotten notes.

**Risks & dependencies.** We need to track updates efficiently (index rebuilds can already be expensive on large vaults) and avoid blocking the TUI while graph metrics load.

**Tasks.**
- [ ] Prototype a lightweight graph data structure derived from the search index documents (e.g., adjacency lists keyed by note path).
- [ ] Extend the index build process to persist backlink metadata and expose a `Related(path string)` helper.
- [ ] Add a new Bubble Tea view or panel that visualizes backlinks and link counts for the focused note.
- [ ] Write unit tests around the new graph helpers to ensure backlinks stay in sync during rebuilds.

## 2. Capture & triage automations

**Vision.** Reduce friction when adding new material by enabling richer capture inputs (templates, clipboard detection, quick tags) and post-capture triage flows.

**Why now.** The current quick capture (scratch buffer) drops markdown into the vault but leaves organization manual. Enhancing automation would streamline inbox processing and make the CLI friendlier to newcomers.

**Risks & dependencies.** Clipboard and OS integrations differ across platforms. We'll need to gracefully degrade when dependencies are missing.

**Tasks.**
- [x] Design a template manifest format (YAML or TOML) and parser to populate default note scaffolds.
- [x] Implement a `an capture --template <name>` subcommand that uses the manifest and still respects configured editors.
- [ ] Add optional auto-tagging rules (e.g., based on capture source) that update front matter before the file hits disk.
- [x] Document the capture workflow in the README with examples and troubleshooting tips.

## 3. Task-focused agenda mode

**Vision.** Elevate the CLI's lightweight task tracking by introducing a dedicated agenda board that aggregates open tasks, due dates, and pinned items across the vault.

**Why now.** We already expose pinned tasks and note metadata, but users still jump to external systems for planning. An agenda mode would keep daily review entirely inside the terminal.

**Risks & dependencies.** Requires consistent parsing of task syntax (checkboxes, due dates) and may need caching to stay responsive on large vaults.

**Tasks.**
- [ ] Expand the parser utilities to detect task states (`[ ]`, `[x]`, due dates) and expose structured results.
- [ ] Build a service that aggregates tasks from configured subdirectories and caches results for quick refresh.
- [ ] Create a Bubble Tea agenda view with filtering (today, upcoming, overdue) and quick actions (mark done, open note).
- [ ] Add integration tests covering task parsing and agenda rendering to protect against regressions.

## 4. Accelerate contextual review experiences

**Vision.** Help users revisit the right knowledge at the right moment by pairing the existing search and templating engines with intelligent resurfacing queues and guided rituals. Review mode should feel purposeful, not like paging through static note lists.

**Why now.** The TUI already supports inline editing, quick capture, and flexible view switching, but review sessions still demand manual navigation. We can unlock more value from the current search index (`internal/search`) and template engine (`pkg/templates`) by repurposing them for scheduled check-ins and contextual discovery.

**Risks & dependencies.** Resurfacing logic depends on accurate timestamps and index freshness, so we need to avoid expensive rebuilds during reviews. Visualization panels add rendering complexity to the Bubble Tea layout, and guided flows must stay responsive even as they orchestrate templates, search filters, and task state.

**Tasks.**
- [ ] Ship a `ReviewQueue` builder that layers modification timestamps onto existing search filters so users can resurface notes last touched N days ago or during specific projects.
- [ ] Persist queue snapshots in the search index to avoid redundant scans and add unit tests that confirm deterministic ordering and correct handling of missing metadata.
- [ ] Add a Bubble Tea visualization panel that renders a backlinks graph using the indexed link data, including hoverable counts and quick navigation actions for linked notes.
- [ ] Expose the graph panel as an optional split view within review mode and ensure layout performance stays responsive on large vaults.
- [ ] Implement guided review flows that step through template-backed checklists (daily, weekly, project retro) and surface contextual quick actions powered by search results.
- [ ] Document how to customize checklist templates and queue presets so review rituals remain repeatable for different teams.

## Operating cadence

To make steady progress, tackle one feature area per iteration, moving a single checkboxed task into "in progress" at a time. After each task lands, revisit the vision statement to ensure the implementation still supports the overarching goal before proceeding to the next task.
