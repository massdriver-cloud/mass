package templatecache

type TemplateCache interface {
	RefreshTemplates() error
	ListTemplates() ([]TemplateList, error)
	GetTemplatePath() (string, error)
}

type Fetcher func(writePath string) error
