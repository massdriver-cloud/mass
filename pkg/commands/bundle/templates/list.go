package templates

import masstemplates "github.com/massdriver-cloud/mass/pkg/templates"

func RunList(repo masstemplates.Repository) ([]string, error) {
	return repo.List()
}
