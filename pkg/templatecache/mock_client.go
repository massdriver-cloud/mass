package templatecache

func NewMockClient(rootTemplateDir string) TemplateCache {
	return &BundleTemplateCache{
		TemplatePath: rootTemplateDir,
	}
}
