package arg

func HandleContent(args []string) string {
	content := ""

	if len(args) > 1 {
		content = args[2]
	}

	return content
}
