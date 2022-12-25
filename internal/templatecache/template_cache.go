package templatecache

type TemplateCache interface {
	RefreshTemplates() error
	ListTemplates() ([]TemplateList, error)
	GetTemplatePath() (string, error)
	RenderTemplate(*TemplateData) error
}

type TemplateData struct {
	Name           string
	Description    string
	Access         string
	Location       string
	TemplateName   string
	TemplateRepo   string
	TemplateSource string
	OutputDir      string
	Type           string
	Connections    map[string]string
	// Specificaly for the README
	CloudPrefix     string
	RepoName        string
	RepoNameEncoded string
}

type Fetcher func(writePath string) error
