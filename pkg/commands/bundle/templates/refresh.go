package templates

import "github.com/massdriver-cloud/mass/pkg/templatecache"

func RunRefresh(cache templatecache.TemplateCache) error {
	return cache.RefreshTemplates()
}
