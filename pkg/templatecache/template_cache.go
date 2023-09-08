package templatecache

type TemplateCache interface {
	RefreshTemplates() error
	ListTemplates() ([]TemplateList, error)
	GetTemplatePath() (string, error)
	RenderTemplate(*TemplateData) error
}

type TemplateData struct {
	Name           string       `json:"name"`
	Description    string       `json:"description"`
	Access         string       `json:"access"`
	Location       string       `json:"location"`
	TemplateName   string       `json:"templateName"`
	TemplateRepo   string       `json:"templateRepo"`
	TemplateSource string       `json:"templateSource"`
	OutputDir      string       `json:"outputDir"`
	Type           string       `json:"type"`
	Connections    []Connection `json:"connections"`
	// Specificaly for the README
	CloudAbbreviation string `json:"cloudAbbreviation"`
	RepoName          string `json:"repoName"`
	RepoNameEncoded   string `json:"repoNameEncoded"`
}

type Connection struct {
	Name               string `json:"name"`
	ArtifactDefinition string `json:"artifact_definition"`
}

type Fetcher func(writePath string) error
