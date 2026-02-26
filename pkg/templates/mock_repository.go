package templates

func NewMockRepository(rootTemplateDir string) Repository {
	return &LocalRepository{
		TemplatePath: rootTemplateDir,
	}
}
