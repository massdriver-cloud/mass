package templates

import "github.com/massdriver-cloud/mass/pkg/templatecache"

func RunList(cache templatecache.TemplateCache) ([]templatecache.TemplateList, error) {
	return cache.ListTemplates()
}
