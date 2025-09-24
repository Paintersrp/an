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
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/Paintersrp/an/internal/note"
	"github.com/Paintersrp/an/internal/state"
	"github.com/Paintersrp/an/internal/templater"
	"github.com/Paintersrp/an/utils"
)

// NewCmdCapture constructs the interactive capture command.
func NewCmdCapture(s *state.State) *cobra.Command {
	var templateName string
	var title string
	var skipPreview bool
	var viewName string

	cmd := &cobra.Command{
		Use:   "capture",
		Short: "Run a guided capture flow for a new note",
		Long: heredoc.Doc(`
                        Capture walks you through selecting a template, reviewing its preview, and collecting
                        the metadata required before the note hits disk. It is ideal for quick captures where you
                        want consistent front matter (status, effort, views) without memorising every flag.
                `),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runCapture(s, templateName, title, viewName, skipPreview)
		},
	}

	cmd.Flags().StringVarP(&templateName, "template", "t", "", "Template to capture with")
	cmd.Flags().StringVarP(&title, "title", "T", "", "Title to assign to the captured note")
	cmd.Flags().BoolVar(&skipPreview, "no-preview", false, "Skip showing the template preview")
	cmd.Flags().StringVar(&viewName, "view", "", "Preselect the view metadata value without prompting")

	return cmd
}

func runCapture(s *state.State, templateName, title, viewName string, skipPreview bool) error {
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

	vaultDir := viper.GetString("vaultdir")
	subDir := viper.GetString("subdir")

	captureNote := note.NewZettelkastenNote(vaultDir, subDir, noteTitle, tags, links, upstream)

	if err := captureNote.HandleConflicts(); err != nil {
		return err
	}

	note.StaticHandleNoteLaunch(captureNote, s.Templater, templateChoice, "", metadata)
	return nil
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
