package artifact

type Artifact struct {
	Data  map[string]interface{} `json:"data"`
	Specs map[string]interface{} `json:"specs"`
}
