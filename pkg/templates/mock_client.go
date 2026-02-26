package templates

func NewMockClient(rootTemplateDir string) TemplateCache {
	return &BundleTemplateCache{
		TemplatePath: rootTemplateDir,
	}
}
