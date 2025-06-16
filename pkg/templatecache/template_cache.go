package templatecache

type TemplateCache interface {
	RefreshTemplates() error
	ListTemplates() ([]TemplateList, error)
	GetTemplatePath() (string, error)
	RenderTemplate(*TemplateData) error
}

type TemplateData struct {
	Name         string            `json:"name"`
	Description  string            `json:"description"`
	Location     string            `json:"location"`
	TemplateName string            `json:"templateName"`
	TemplateRepo string            `json:"templateRepo"`
	OutputDir    string            `json:"outputDir"`
	Type         string            `json:"type"`
	Connections  []Connection      `json:"connections"`
	Envs         map[string]string `json:"envs"`

	// ParamsSchema is a YAML formatted string
	ParamsSchema string `json:"paramsSchema"`

	// Path to a terraform-module or helm-chart to parse for params
	ExistingParamsPath string `json:"existingParamsPath"`
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
