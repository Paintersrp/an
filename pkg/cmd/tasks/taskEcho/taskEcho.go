package taskEcho

import (
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/Paintersrp/an/internal/state"
)

// TODO: Clean

func NewCmdTaskEcho(s *state.State) *cobra.Command {
	var priority string
	var target string

	cmd := &cobra.Command{
		Use:     "echo [task] -p {priority} -t {target}",
		Aliases: []string{"e"},
		Short:   "Append a task to the pinned task file with optional priority.",
		Long: `The task-echo command appends a task to the pinned task file under the "## Tasks" section.
It allows for tasks to be categorized under high, medium, or low priority sections.`,
		Example: `
    # Echo a task with high priority
    an-cli tasks echo "Finish the report" -p high
    `,
		Args: cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return run(args, s, priority, target)
		},
	}

	cmd.Flags().
		StringVarP(&priority, "priority", "p", "low", "Priority of the task (high, medium, low).")
	cmd.Flags().
		StringVarP(&target, "target", "t", "", "Named task pin to target.")

	return cmd
}

func run(args []string, s *state.State, priority string, target string) error {
	task := strings.Join(args, " ")
	taskEntry := fmt.Sprintf("- [ ] %s\n", task)
	var targetPin string

	if target != "" {
		if s.Config.NamedTaskPins[target] == "" {
			return fmt.Errorf(
				"no task file pinned for named task pin '%s'. Use the task-pin command to pin a task-file first",
				target,
			)
		}
		targetPin = s.Config.NamedTaskPins[target]
	} else {
		if s.Config.PinnedTaskFile == "" {
			return errors.New(
				"no task file pinned. Use the task-pin command to pin a task-file first",
			)
		}
		targetPin = s.Config.PinnedTaskFile
	}

	// Read the entire file into memory
	content, err := os.ReadFile(targetPin)
	if err != nil {
		return err
	}

	// Convert content to a string and check for the "## Tasks" section
	contentStr := string(content)
	taskSection := "## Tasks\n"
	if !strings.Contains(contentStr, taskSection) {
		// If "## Tasks" section doesn't exist, add it to the end of the file
		contentStr += "\n" + taskSection
	}

	// Define the priority sections
	prioritySections := map[string]string{
		"low":    "### Low Priority\n",
		"medium": "### Medium Priority\n",
		"high":   "### High Priority\n",
	}

	// Check for the existence of priority sections and insert if missing
	for _, prio := range []string{"low", "medium", "high"} {
		sec := prioritySections[prio]
		if !strings.Contains(contentStr, sec) {
			index := strings.Index(contentStr, taskSection) + len(taskSection)
			contentStr = contentStr[:index] + sec + contentStr[index:]
		}
	}

	// Determine where to place the task based on priority
	section := prioritySections[priority]
	if section == "" {
		section = prioritySections["low"] // Default to low priority if not specified
	}

	// Find the index to insert the task
	sectionIndex := strings.Index(contentStr, section) + len(section)
	nextSectionIndex := strings.Index(contentStr[sectionIndex:], "###")
	if nextSectionIndex == -1 {
		nextSectionIndex = len(contentStr)
	} else {
		nextSectionIndex += sectionIndex
	}

	// Find the end of the current section to insert the task after existing tasks
	endOfSectionIndex := strings.LastIndex(
		contentStr[:nextSectionIndex],
		"\n",
	) + 1
	if endOfSectionIndex < sectionIndex { // If no tasks in the section, set to sectionIndex
		endOfSectionIndex = sectionIndex
	}

	// Insert the task in the correct position
	contentStr = contentStr[:endOfSectionIndex] + taskEntry + contentStr[endOfSectionIndex:]

	// Write the updated content back to the file
	err = os.WriteFile(targetPin, []byte(contentStr), 0644)
	if err != nil {
		return err
	}

	if target != "" {
		fmt.Printf(
			"Task appended to the pinned named task file '%s' under the \"%s\" section.\n",
			target,
			priority,
		)
	} else {
		fmt.Printf(
			"Task appended to the pinned task file under the \"%s\" section.\n",
			priority,
		)
	}
	return nil

}
