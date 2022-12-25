package commands

import (
	"github.com/massdriver-cloud/mass/internal/templatecache"
)

func GenerateNewBundle(bundleCache templatecache.TemplateCache, templateData *templatecache.TemplateData) error {
	return bundleCache.RenderTemplate(templateData)
}
