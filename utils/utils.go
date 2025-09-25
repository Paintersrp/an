package utils

import (
	"fmt"
	"io"
	"math"
	"os"
	"regexp"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/charmbracelet/glamour"
	"github.com/charmbracelet/glamour/ansi"
	"github.com/muesli/termenv"
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/renderer"
	"github.com/yuin/goldmark/util"
)

func AppendIfNotExists(slice []string, value string) []string {
	for _, v := range slice {
		if v == value {
			return slice
		}
	}
	return append(slice, value)
}

func ValidateInput(input string) ([]string, error) {
	if input == "" {
		return []string{}, nil
	}

	items := strings.Split(input, " ")
	for _, item := range items {
		if !isValidInput(item) {
			return nil, fmt.Errorf(
				"invalid input '%s': Input must only contain alphanumeric characters, hyphens, and underscores",
				item,
			)
		}
	}
	return items, nil
}

func isValidInput(input string) bool {
	return regexp.MustCompile(`^[a-zA-Z0-9-_]+$`).MatchString(input)
}

func GenerateDate(numUnits int, unitType string) string {
	var date time.Time
	var dateFormat string
	now := time.Now()

	switch unitType {
	case "day":
		date = now.AddDate(0, 0, numUnits)
		dateFormat = "20060102"
	case "week":
		offset := int(time.Sunday - now.Weekday())
		if offset > 0 {
			offset = -6
		}
		startOfWeek := now.AddDate(0, 0, offset)
		date = startOfWeek.AddDate(0, 0, numUnits*7)
		dateFormat = "20060102"
	case "month":
		startOfMonth := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())
		date = startOfMonth.AddDate(0, numUnits, 0)
		dateFormat = "200601"
	case "year":
		startOfYear := time.Date(now.Year(), 1, 1, 0, 0, 0, 0, now.Location())
		date = startOfYear.AddDate(numUnits, 0, 0)
		dateFormat = "2006"
	default:
		date = now
		dateFormat = "20060102"
	}

	return date.Format(dateFormat)
}

func ParseFrontmatter(content string) (string, string) {
	frontmatterRegex := regexp.MustCompile(`(?s)\A---\r?\n(.*?)\r?\n---\r?\n?`)
	matches := frontmatterRegex.FindStringSubmatch(content)

	var frontmatter, markdown string
	if len(matches) > 1 {
		frontmatter = matches[1]
		markdown = strings.TrimPrefix(content, matches[0])
	} else {
		markdown = content
	}

	return frontmatter, markdown
}

func FormatFrontmatterAsMarkdown(frontmatter string) string {
	lines := strings.Split(frontmatter, "\n")
	formattedLines := make([]string, 0, len(lines))

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.Contains(line, ":") {
			parts := strings.SplitN(line, ":", 2)
			key := strings.TrimSpace(parts[0])
			value := strings.TrimSpace(parts[1])

			if value != "" {
				formattedLines = append(
					formattedLines,
					fmt.Sprintf("**%s:** %s", key, value),
				)
			}
		} else if line != "" {
			formattedLines = append(formattedLines, line)
		}
	}

	return strings.Join(formattedLines, "\n\n")
}

func RenderMarkdownPreview(path string, w, h int) string {
	renderedContent, err := BuildMarkdownPreviewContent(path)
	if err != nil {
		return "Error reading file"
	}

	var builder strings.Builder
	if err := RenderMarkdownContent(renderedContent, w, &builder); err != nil {
		return "Error rendering markdown"
	}

	return builder.String()
}

func BuildMarkdownPreviewContent(path string) (string, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}

	frontmatter, markdown := ParseFrontmatter(string(content))
	formattedFrontmatter := FormatFrontmatterAsMarkdown(frontmatter)

	if formattedFrontmatter != "" {
		return formattedFrontmatter + "\n\n---\n\n\n" + markdown, nil
	}

	return "No frontmatter found.\n\n---\n\n\n" + markdown, nil
}

func RenderMarkdownContent(content string, width int, writer io.Writer) error {
	wordWrap := width
	if wordWrap <= 0 {
		wordWrap = 100
	}

	options := ansi.Options{
		WordWrap:     wordWrap,
		ColorProfile: termenv.ANSI256,
		Styles:       glamour.DraculaStyleConfig,
	}

	md := goldmark.New(
		goldmark.WithExtensions(
			extension.GFM,
			extension.DefinitionList,
		),
		goldmark.WithParserOptions(
			parser.WithAutoHeadingID(),
		),
		goldmark.WithRenderer(
			renderer.NewRenderer(
				renderer.WithNodeRenderers(
					util.Prioritized(ansi.NewRenderer(options), 1000),
				),
			),
		),
	)

	bw := &passthroughWriter{w: writer}
	if err := md.Convert([]byte(content), bw); err != nil {
		return err
	}

	return nil
}

type passthroughWriter struct {
	w io.Writer
}

func (p *passthroughWriter) Write(b []byte) (int, error) {
	return p.w.Write(b)
}

func (p *passthroughWriter) WriteString(s string) (int, error) {
	return io.WriteString(p.w, s)
}

func (p *passthroughWriter) WriteByte(c byte) error {
	_, err := p.w.Write([]byte{c})
	return err
}

func (p *passthroughWriter) WriteRune(r rune) (int, error) {
	var buf [utf8.UTFMax]byte
	n := utf8.EncodeRune(buf[:], r)
	_, err := p.w.Write(buf[:n])
	return n, err
}

func (p *passthroughWriter) Flush() error {
	return nil
}

func (p *passthroughWriter) Available() int {
	return math.MaxInt32
}

func (p *passthroughWriter) Buffered() int {
	return 0
}

func FormatBytes(size int64) string {
	units := []string{"B", "KB", "MB", "GB", "TB", "PB", "EB"}
	var mod int64 = 1024
	var i int
	for size >= mod {
		size /= mod
		i++
	}
	return fmt.Sprintf("%d %s", size, units[i])
}
