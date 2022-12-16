package commands

import "github.com/massdriver-cloud/mass/internal/templatecache"

func ListTemplates(cache templatecache.TemplateCache) ([]string, error) {
	return cache.ListTemplates()
}
