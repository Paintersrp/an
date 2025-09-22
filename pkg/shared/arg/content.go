package arg

func HandleContent(args []string) string {
	content := ""

	if len(args) >= 3 {
		content = args[2]
	}

	return content
}
