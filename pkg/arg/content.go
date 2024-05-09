package arg

func HandleContent(args []string) string {
	var content = ""

	if len(args) > 1 {
		content = args[2]
	}

	return content
}
