package artifact

type Artifact struct {
	Data  map[string]any `json:"data"`
	Specs map[string]any `json:"specs"`
}
