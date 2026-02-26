package templates

import masstemplates "github.com/massdriver-cloud/mass/pkg/templates"

func RunList(t *masstemplates.Templates) ([]string, error) {
	return t.List()
}
