package sync

type Frontmatter struct {
	Title       string   `yaml:"title"`
	Upstream    string   `yaml:"up"`
	CreatedAt   string   `yaml:"created"`
	Tags        []string `yaml:"tags"`
	LinkedNotes []string
}

type NotePayload struct {
	Title    string   `json:"title"       validate:"required"`
	NewTitle string   `json:"new_title"`
	Tags     []string `json:"tags"`
	UserID   int32    `json:"user_id"     validate:"required"`
	VaultID  int32    `json:"vault_id"    validate:"required"`
	Upstream *int32   `json:"upstream_id"`
	Links    []int32  `json:"links"`
	Content  string   `json:"content"     validate:"required"`
}

type NoteDeletePayload struct {
	UserID int32  `json:"user_id" validate:"required"`
	Title  string `json:"title"   validate:"required"`
}

type NoteOperation struct {
	Operation     string             `json:"operation"`
	UpdatePayload *NotePayload       `json:"update_payload,omitempty"`
	DeletePayload *NoteDeletePayload `json:"delete_payload,omitempty"`
}

type BulkNoteOperationPayload struct {
	Operations []NoteOperation `json:"operations"`
}
