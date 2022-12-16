package commands

import "github.com/massdriver-cloud/mass/internal/template_cache"

func ListTemplates(cache template_cache.TemplateCache) ([]string, error) {
	return cache.ListTemplates()
}
