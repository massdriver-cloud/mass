package templatecache

type TemplateCache interface {
	RefreshTemplates() error
	ListTemplates() ([]string, error)
	GetTemplatePath() (string, error)
}

type Fetcher func(writePath string) error
