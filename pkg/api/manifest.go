package api

type Manifest struct {
	ID     string `json:"id"`
	Bundle Bundle `json:"bundle"`
}
