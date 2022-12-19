package commands

import "github.com/massdriver-cloud/mass/internal/templatecache"

func ListTemplates(cache templatecache.TemplateCache) ([]templatecache.TemplateList, error) {
	return cache.ListTemplates()
}
