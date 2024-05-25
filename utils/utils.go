package utils

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/charmbracelet/glamour"
	"github.com/golang-jwt/jwt"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/muesli/termenv"
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

func ReadFileAndTrimContent(path string, cutoff int) (string, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}

	if len(content) > cutoff {
		content = content[:cutoff]
	}

	return string(content), nil
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
	const cutoff = 1000

	content, err := ReadFileAndTrimContent(path, cutoff)
	if err != nil {
		return "Error reading file"
	}

	frontmatter, markdown := ParseFrontmatter(content)
	formattedFrontmatter := FormatFrontmatterAsMarkdown(frontmatter)

	var renderedContent string
	if formattedFrontmatter != "" {
		renderedContent = formattedFrontmatter + "\n\n---\n\n\n" + markdown
	} else {
		renderedContent = "No frontmatter found.\n\n---\n\n\n" + markdown
	}

	r, _ := glamour.NewTermRenderer(
		glamour.WithStandardStyle("dracula"),
		glamour.WithWordWrap(100),
		glamour.WithColorProfile(termenv.ANSI256),
	)

	renderedMarkdown, err := r.Render(renderedContent)
	if err != nil {
		return "Error rendering markdown"
	}

	return renderedMarkdown
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

type Claims struct {
	jwt.StandardClaims
	UserID   int64       `json:"user_id"`
	Username string      `json:"username"`
	Email    string      `json:"email"`
	RoleID   pgtype.Int4 `json:"role_id"`
}

func GetClaims(tokenString, secret string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(
		tokenString,
		&Claims{},
		func(token *jwt.Token) (interface{}, error) {
			return []byte(secret), nil
		},
	)
	if err != nil {
		return nil, err
	}

	if claims, ok := token.Claims.(*Claims); ok && token.Valid {
		return claims, nil
	}

	return nil, err
}

func IsValidDirName(name string) error {
	if strings.TrimSpace(name) == "" {
		return errors.New("directory name cannot be empty")
	}
	if name != filepath.Clean(name) {
		return errors.New("directory name contains invalid characters or sequence")
	}
	if strings.Contains(name, string(filepath.Separator)) {
		return errors.New("directory name must not contain path separators")
	}
	return nil
}

type VaultRecord struct {
	ID        int32
	Name      string
	UserID    int32
	Commit    string
	CreatedAt time.Time
	UpdatedAt time.Time
}

func SendVaultCreateRequest(
	token,
	vaultName string,
	claims *Claims,
) (*VaultRecord, error) {
	data := map[string]interface{}{
		"name":    vaultName,
		"user_id": claims.UserID,
	}

	dataJson, err := json.Marshal(data)
	if err != nil {
		fmt.Printf("failed to encode data to JSON: %v", err)
		return nil, err
	}

	req, err := http.NewRequest(
		"POST",
		"http://localhost:6474/v1/api/vaults",
		bytes.NewBuffer(dataJson),
	)
	if err != nil {
		fmt.Printf("failed to create request: %v", err)
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		fmt.Printf("failed to send request: %v", err)
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		return nil, fmt.Errorf("status code: %d", resp.StatusCode)
	}

	var respData VaultRecord
	err = json.NewDecoder(resp.Body).Decode(&respData)
	if err != nil {
		fmt.Printf("failed to decode response: %v", err)
		return nil, err
	}

	return &respData, nil
}

func GetVaultByName(
	token,
	vaultName string,
) (*VaultRecord, error) {
	data := map[string]interface{}{
		"name": vaultName,
	}

	dataJson, err := json.Marshal(data)
	if err != nil {
		fmt.Printf("failed to encode data to JSON: %v", err)
		return nil, err
	}

	req, err := http.NewRequest(
		"POST",
		"http://localhost:6474/v1/api/vaults/find",
		bytes.NewBuffer(dataJson),
	)
	if err != nil {
		fmt.Printf("failed to create request: %v", err)
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		fmt.Printf("failed to send request: %v", err)
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		return nil, fmt.Errorf("status code: %d", resp.StatusCode)
	}

	var respData VaultRecord
	err = json.NewDecoder(resp.Body).Decode(&respData)
	if err != nil {
		fmt.Printf("failed to decode response: %v", err)
		return nil, err
	}

	return &respData, nil
}
