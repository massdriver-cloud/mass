package commands

import "github.com/massdriver-cloud/mass/pkg/templatecache"

func RefreshTemplates(cache templatecache.TemplateCache) error {
	return cache.RefreshTemplates()
}
