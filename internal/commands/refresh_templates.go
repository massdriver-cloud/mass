package commands

import "github.com/massdriver-cloud/mass/internal/templatecache"

func RefreshTemplates(cache templatecache.TemplateCache) error {
	return cache.RefreshTemplates()
}
