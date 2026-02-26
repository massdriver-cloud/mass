package templates

import masstemplates "github.com/massdriver-cloud/mass/pkg/templates"

func RunList(cache masstemplates.TemplateCache) ([]string, error) {
	return cache.ListTemplates()
}
