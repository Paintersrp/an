package capture

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"sort"
	"strconv"
	"strings"

	"github.com/MakeNowJust/heredoc/v2"
	"github.com/atotto/clipboard"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/Paintersrp/an/internal/config"
	"github.com/Paintersrp/an/internal/note"
	"github.com/Paintersrp/an/internal/state"
	"github.com/Paintersrp/an/internal/templater"
	"github.com/Paintersrp/an/utils"
)

var readClipboard = clipboard.ReadAll

// NewCmdCapture constructs the interactive capture command.
func NewCmdCapture(s *state.State) *cobra.Command {
	var templateName string
	var title string
	var skipPreview bool
	var viewName string
	var dryRun bool

	cmd := &cobra.Command{
		Use:   "capture",
		Short: "Run a guided capture flow for a new note",
		Long: heredoc.Doc(`
                        Capture walks you through selecting a template, reviewing its preview, and collecting
                        the metadata required before the note hits disk. It is ideal for quick captures where you
                        want consistent front matter (status, effort, views) without memorising every flag.
                `),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runCapture(s, templateName, title, viewName, skipPreview, dryRun)
		},
	}

	cmd.Flags().StringVarP(&templateName, "template", "t", "", "Template to capture with")
	cmd.Flags().StringVarP(&title, "title", "T", "", "Title to assign to the captured note")
	cmd.Flags().BoolVar(&skipPreview, "no-preview", false, "Skip showing the template preview")
	cmd.Flags().StringVar(&viewName, "view", "", "Preselect the view metadata value without prompting")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "Preview metadata without creating a note")

	return cmd
}

func runCapture(s *state.State, templateName, title, viewName string, skipPreview, dryRun bool) error {
	if s == nil || s.Templater == nil {
		return errors.New("capture requires an initialised workspace and templater")
	}

	reader := bufio.NewReader(os.Stdin)

	templateChoice, err := selectTemplate(reader, s.Templater, templateName)
	if err != nil {
		return err
	}

	if !skipPreview {
		if err := showPreview(s.Templater, templateChoice); err != nil {
			return err
		}
	}

	noteTitle, err := ensureTitle(reader, title)
	if err != nil {
		return err
	}

	tags, err := promptList(reader, "Tags (space separated, optional)")
	if err != nil {
		return err
	}

	links, err := promptList(reader, "Links (space separated, optional)")
	if err != nil {
		return err
	}

	upstream, err := promptValue(reader, "Upstream note (optional)")
	if err != nil {
		return err
	}

	viewSelection, err := resolveView(reader, s, viewName)
	if err != nil {
		return err
	}

	metadata, err := note.CollectTemplateMetadata(s.Templater, templateChoice)
	if err != nil {
		return err
	}
	if metadata == nil {
		metadata = make(map[string]interface{})
	}
	if viewSelection != "" {
		metadata["view"] = viewSelection
	}

	ruleTags, ruleMetadata, err := resolveCaptureMetadata(s, templateChoice, upstream)
	if err != nil {
		return err
	}

	tags = mergeTagSets(tags, ruleTags)
	metadata = mergeMetadata(metadata, ruleMetadata)

	if err := maybePreviewCaptureMetadata(reader, tags, metadata, dryRun); err != nil {
		return err
	}

	if dryRun {
		return nil
	}

	vaultDir := viper.GetString("vaultdir")
	subDir := viper.GetString("subdir")

	captureNote := note.NewZettelkastenNote(vaultDir, subDir, noteTitle, tags, links, upstream)

	if err := captureNote.HandleConflicts(); err != nil {
		return err
	}

	note.StaticHandleNoteLaunch(captureNote, s.Templater, templateChoice, "", metadata)
	return nil
}

func resolveCaptureMetadata(s *state.State, templateName, upstream string) ([]string, map[string]any, error) {
	if s == nil || s.Workspace == nil {
		return nil, nil, nil
	}

	var (
		overlayTags    []string
		frontMatter    map[string]any
		clipboardValue string
		clipboardRead  bool
	)

	for _, rule := range s.Workspace.Capture.Rules {
		if !matchesTemplate(rule, templateName) {
			continue
		}
		if !matchesUpstream(rule, upstream) {
			continue
		}
		if rule.Action.Clipboard {
			if !clipboardRead {
				value, err := readClipboard()
				if err != nil {
					return nil, nil, fmt.Errorf("read clipboard: %w", err)
				}
				clipboardValue = value
				clipboardRead = true
			}
			if strings.TrimSpace(clipboardValue) == "" {
				continue
			}
		}

		for _, tag := range rule.Action.Tags {
			tag = strings.TrimSpace(tag)
			if tag == "" {
				continue
			}
			overlayTags = append(overlayTags, tag)
		}

		if len(rule.Action.FrontMatter) > 0 {
			if frontMatter == nil {
				frontMatter = make(map[string]any)
			}
			for key, value := range rule.Action.FrontMatter {
				frontMatter[key] = value
			}
		}
	}

	overlayTags = dedupePreserveOrder(overlayTags)

	if len(frontMatter) == 0 {
		frontMatter = nil
	}

	return overlayTags, frontMatter, nil
}

func matchesTemplate(rule config.CaptureRule, template string) bool {
	if rule.Match.Template == "" {
		return true
	}
	return rule.Match.Template == template
}

func matchesUpstream(rule config.CaptureRule, upstream string) bool {
	if rule.Match.UpstreamPrefix == "" {
		return true
	}
	return strings.HasPrefix(upstream, rule.Match.UpstreamPrefix)
}

func dedupePreserveOrder(values []string) []string {
	if len(values) == 0 {
		return nil
	}
	seen := make(map[string]struct{}, len(values))
	result := make([]string, 0, len(values))
	for _, value := range values {
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		result = append(result, value)
	}
	return result
}

func mergeTagSets(base, overlays []string) []string {
	if len(overlays) == 0 {
		return base
	}

	seen := make(map[string]struct{}, len(base)+len(overlays))
	merged := make([]string, 0, len(base)+len(overlays))

	for _, tag := range base {
		if _, ok := seen[tag]; ok {
			continue
		}
		seen[tag] = struct{}{}
		merged = append(merged, tag)
	}

	for _, tag := range overlays {
		if _, ok := seen[tag]; ok {
			continue
		}
		seen[tag] = struct{}{}
		merged = append(merged, tag)
	}

	return merged
}

func mergeMetadata(base map[string]interface{}, overlays map[string]any) map[string]interface{} {
	if len(overlays) == 0 {
		return base
	}

	result := make(map[string]interface{}, len(base)+len(overlays))
	for key, value := range base {
		result[key] = value
	}
	for key, value := range overlays {
		result[key] = value
	}

	return result
}

func maybePreviewCaptureMetadata(reader *bufio.Reader, tags []string, metadata map[string]interface{}, dryRun bool) error {
	if dryRun {
		printCaptureMetadataPreview(tags, metadata)
		return nil
	}

	show, err := promptYesNo(reader, "Show metadata preview? (y/N): ")
	if err != nil {
		return err
	}
	if show {
		printCaptureMetadataPreview(tags, metadata)
	}
	return nil
}

func printCaptureMetadataPreview(tags []string, metadata map[string]interface{}) {
	fmt.Println("\nCapture metadata preview:")
	if len(tags) == 0 {
		fmt.Println("  Tags: (none)")
	} else {
		fmt.Printf("  Tags: %s\n", strings.Join(tags, ", "))
	}

	if len(metadata) == 0 {
		fmt.Println("  Front matter: (none)")
		return
	}

	fmt.Println("  Front matter:")
	keys := make([]string, 0, len(metadata))
	for key := range metadata {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		fmt.Printf("    %s: %v\n", key, metadata[key])
	}
}

func promptYesNo(reader *bufio.Reader, label string) (bool, error) {
	for {
		fmt.Print(label)
		raw, err := reader.ReadString('\n')
		if err != nil {
			return false, err
		}
		raw = strings.TrimSpace(strings.ToLower(raw))
		switch raw {
		case "", "n", "no":
			return false, nil
		case "y", "yes":
			return true, nil
		default:
			fmt.Println("Please enter 'y' or 'n'.")
		}
	}
}

func selectTemplate(reader *bufio.Reader, t *templater.Templater, explicit string) (string, error) {
	if explicit != "" {
		if _, err := t.Manifest(explicit); err != nil {
			return "", fmt.Errorf("unknown template %q: %w", explicit, err)
		}
		return explicit, nil
	}

	templates := t.Templates()
	if len(templates) == 0 {
		return "", errors.New("no templates available")
	}

	fmt.Println("Available templates:")
	for idx, name := range templates {
		fmt.Printf("  %d. %s\n", idx+1, name)
	}

	for {
		fmt.Print("Choose a template by number: ")
		choiceRaw, _ := reader.ReadString('\n')
		choiceRaw = strings.TrimSpace(choiceRaw)
		if choiceRaw == "" {
			fmt.Printf("Please select a template.\n")
			continue
		}
		idx, err := parseIndex(choiceRaw, len(templates))
		if err != nil {
			fmt.Printf("%v\n", err)
			continue
		}
		return templates[idx], nil
	}
}

func showPreview(t *templater.Templater, name string) error {
	preview, err := t.Preview(name)
	if err != nil {
		return err
	}
	fmt.Printf("\n--- %s preview ---\n%s\n------------------------\n\n", name, preview)
	return nil
}

func ensureTitle(reader *bufio.Reader, provided string) (string, error) {
	if strings.TrimSpace(provided) != "" {
		return provided, nil
	}

	for {
		fmt.Print("Note title: ")
		title, _ := reader.ReadString('\n')
		title = strings.TrimSpace(title)
		if title == "" {
			fmt.Println("Title is required for capture.")
			continue
		}
		return title, nil
	}
}

func promptList(reader *bufio.Reader, label string) ([]string, error) {
	fmt.Printf("%s: ", label)
	value, _ := reader.ReadString('\n')
	value = strings.TrimSpace(value)
	if value == "" {
		return []string{}, nil
	}
	parsed, err := utils.ValidateInput(value)
	if err != nil {
		return nil, err
	}
	return parsed, nil
}

func promptValue(reader *bufio.Reader, label string) (string, error) {
	fmt.Printf("%s: ", label)
	value, _ := reader.ReadString('\n')
	return strings.TrimSpace(value), nil
}

func resolveView(reader *bufio.Reader, s *state.State, provided string) (string, error) {
	if s == nil || s.Workspace == nil {
		return provided, nil
	}
	views := make([]string, 0, len(s.Workspace.Views))
	for name := range s.Workspace.Views {
		views = append(views, name)
	}
	sort.Strings(views)
	if len(views) == 0 {
		return provided, nil
	}
	if provided != "" {
		if !contains(views, provided) {
			return "", fmt.Errorf("view %q is not defined in this workspace", provided)
		}
		return provided, nil
	}

	fmt.Println("Available views:")
	fmt.Println("  0. (skip)")
	for idx, name := range views {
		fmt.Printf("  %d. %s\n", idx+1, name)
	}

	for {
		fmt.Print("Assign to view: ")
		raw, _ := reader.ReadString('\n')
		raw = strings.TrimSpace(raw)
		if raw == "" || raw == "0" {
			return "", nil
		}
		idx, err := parseIndex(raw, len(views))
		if err != nil {
			fmt.Printf("%v\n", err)
			continue
		}
		return views[idx], nil
	}
}

func parseIndex(value string, length int) (int, error) {
	parsed, err := strconv.Atoi(value)
	if err != nil {
		return 0, fmt.Errorf("please enter a number between 1 and %d", length)
	}
	if parsed <= 0 || parsed > length {
		return 0, fmt.Errorf("please enter a number between 1 and %d", length)
	}
	return parsed - 1, nil
}

func contains(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}
