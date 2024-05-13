package pinList

type PinListItem struct {
	title       string
	description string
}

func (i PinListItem) Title() string       { return i.title }
func (i PinListItem) Description() string { return i.description }
func (i PinListItem) FilterValue() string { return i.title }
