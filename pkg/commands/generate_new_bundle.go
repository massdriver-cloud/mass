package commands

import (
	"github.com/massdriver-cloud/mass/pkg/templatecache"
)

func GenerateNewBundle(bundleCache templatecache.TemplateCache, templateData *templatecache.TemplateData) error {
	return bundleCache.RenderTemplate(templateData)
}
