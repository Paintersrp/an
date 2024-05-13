package arg

import "fmt"

func HandleTitle(args []string) (string, error) {
	if len(args) == 0 {
		return "", fmt.Errorf(
			"error: No title given. Try again",
		)
	}
	return args[0], nil

}
