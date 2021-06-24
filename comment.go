package gopenqa

type Comment struct {
	ID       int      `json:"id"`
	Text     string   `json:"text"`             // Comment text
	Markdown string   `json:"renderedMarkdown"` // HTML of the rendered markdown
	BugRefs  []string `json:"bugrefs"`          // Referenced bugs
	Created  string   `json:"created"`          // bug creation date
	Updated  string   `json:"updated"`          // timestamp for update
	User     string   `json:"userName"`         // Creator
}
