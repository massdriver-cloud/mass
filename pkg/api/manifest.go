package api

type Manifest struct {
	ID          string  `json:"id"`
	Slug        string  `json:"slug"`
	Name        string  `json:"name"`
	Suffix      string  `json:"suffix"`
	Description string  `json:"description"`
	Bundle      *Bundle `json:"bundle"`
}
