package templates

import "github.com/massdriver-cloud/mass/pkg/templatecache"

func RunList(cache templatecache.TemplateCache) ([]string, error) {
	return cache.ListTemplates()
}
