package scalars

// Cursor represents pagination cursor with omitempty on all fields
// to avoid sending empty strings to the server
type Cursor struct {
	Limit    int    `json:"limit,omitempty"`
	Next     string `json:"next,omitempty"`
	Previous string `json:"previous,omitempty"`
}
