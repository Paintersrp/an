package review

import (
	"bufio"
	"fmt"
	"io"
	"strings"
	"unicode/utf8"

	"github.com/Paintersrp/an/internal/templater"
)

// RunChecklist steps through the provided template manifest, prompting the
// reader for responses and displaying contextual resurfacing suggestions to the
// writer.
func RunChecklist(
	manifest templater.TemplateManifest,
	queue []ResurfaceItem,
	reader io.Reader,
	writer io.Writer,
) (map[string]string, error) {
	responses := make(map[string]string)
	if writer == nil {
		writer = io.Discard
	}
	if reader == nil {
		reader = strings.NewReader("")
	}

	bufReader := bufio.NewReader(reader)
	fmt.Fprintf(writer, "\n=== %s checklist ===\n", manifest.Name)

	for idx, field := range manifest.Fields {
		title := field.Label
		if title == "" {
			title = humanizeKey(field.Key)
		}

		fmt.Fprintf(writer, "\nStep %d: %s\n", idx+1, title)
		if field.Prompt != "" {
			fmt.Fprintf(writer, "%s\n", field.Prompt)
		}

		if len(field.Defaults) > 0 {
			fmt.Fprintf(writer, "Suggested focus tags: %s\n", strings.Join(field.Defaults, ", "))
			suggestions := FilterQueue(queue, field.Defaults, nil)
			if len(suggestions) > 0 {
				fmt.Fprintln(writer, "Related resurfacing candidates:")
				for _, item := range suggestions {
					fmt.Fprintf(
						writer,
						"  • %s (last touched %s)\n",
						item.Path,
						item.ModifiedAt.Format("2006-01-02"),
					)
				}
			}
		}

		if len(field.Options) > 0 {
			fmt.Fprintf(writer, "Options: %s\n", strings.Join(field.Options, ", "))
		}

		fmt.Fprint(writer, "> ")
		input, err := bufReader.ReadString('\n')
		if err != nil && err != io.EOF {
			return responses, err
		}
		cleaned := strings.TrimSpace(input)
		responses[field.Key] = cleaned

		if err == io.EOF {
			break
		}
	}

	fmt.Fprintln(writer, "\nChecklist complete. Review notes saved for reference.")
	return responses, nil
}

func humanizeKey(key string) string {
	replaced := strings.ReplaceAll(key, "-", " ")
	replaced = strings.ReplaceAll(replaced, "_", " ")
	if replaced == "" {
		return ""
	}

	parts := strings.Fields(replaced)
	for i, part := range parts {
		if part == "" {
			continue
		}
		r, size := utf8.DecodeRuneInString(part)
		if r == utf8.RuneError && size == 0 {
			continue
		}
		parts[i] = strings.ToUpper(string(r)) + part[size:]
	}
	return strings.Join(parts, " ")
}
