package commands

import "github.com/massdriver-cloud/mass/internal/template_cache"

func RefreshTemplates(cache template_cache.TemplateCache) error {
	return cache.RefreshTemplates()
}
