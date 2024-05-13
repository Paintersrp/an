package sublist

type SubListItem struct {
	title       string
	description string
}

func (i SubListItem) Title() string       { return i.title }
func (i SubListItem) Description() string { return i.description }
func (i SubListItem) FilterValue() string { return i.title }
