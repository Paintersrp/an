package utils

import (
	"fmt"
	"regexp"
	"strings"
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
		return []string{}, nil // No input provided, return an empty slice
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
	// Define the criteria for a valid input, for example:
	// A valid input contains only letters, numbers, hyphens, and underscores.
	return regexp.MustCompile(`^[a-zA-Z0-9-_]+$`).MatchString(input)
}
