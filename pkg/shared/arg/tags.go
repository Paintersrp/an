package arg

import (
	"fmt"
	"os"

	"github.com/Paintersrp/an/utils"
)

func HandleTags(args []string) []string {
	var (
		tags []string
		err  error
	)

	if len(args) > 1 {
		tags, err = utils.ValidateInput(args[1])
		if err != nil {
			fmt.Printf("error processing tags argument: %s", err)
			os.Exit(1)
		}

	}

	return tags
}
