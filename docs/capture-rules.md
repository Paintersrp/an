# Capture rules & automation

Capture rules live inside each workspace configuration in `~/.an/cfg.yaml`. They
run during `an capture` after you answer the interactive prompts but before the
note is written, letting you inject tags or front matter based on the template,
upstream target, or clipboard state.

```yaml
workspaces:
  default:
    vaultdir: /Users/me/notes
    editor: nvim
    capture:
      rules:
        - match:
            template: project-release
          action:
            tags: [release, status/inbox]
            front_matter:
              status: drafted
              reviewer: qa-team
```

The example above applies whenever the `project-release` template is selected.
It overlays two tags and assigns front matter values before the note opens in
your editor.

## Matching strategies

Every rule can include either matcher:

- `match.template` limits the rule to a specific template name. Omit the field
to apply it to every template.
- `match.upstream_prefix` checks the upstream note path you provide to the
capture wizard. Use it to scope rules to projects, e.g. only when linking to
`research/` folders.

If both matchers are present, they must succeed for the rule to fire. Multiple
rules can fire for the same capture and their metadata is merged in the order
they appear in the configuration.

## Clipboard-aware notes

Set `action.clipboard: true` to require a non-empty clipboard before a rule
runs. This is handy for "clip and capture" flows so empty clipboards do not add
noise:

```yaml
    capture:
      rules:
        - match:
            upstream_prefix: inbox/
          action:
            clipboard: true
            tags: [source/clipboard]
            front_matter:
              collected_via: clipboard
```

When the clipboard is empty the rule is skipped, but any other rules that do not
check the clipboard still apply.

## Tag & front matter deduplication

Manual tags entered during capture are merged with rule-driven tags while
preserving order and removing duplicates. Front matter values from later rules
override earlier ones, making it simple to compose defaults with targeted
overrides. Combine these rules with `an capture --dry-run` to preview the final
metadata without writing a file.
