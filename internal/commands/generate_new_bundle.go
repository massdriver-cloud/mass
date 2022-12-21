package commands

import "github.com/spf13/afero"

type TemplateData struct {
	Name           string
	Description    string
	Access         string
	Location       string
	TemplateName   string
	TemplateSource string
	OutputDir      string
	Type           string
	Connections    map[string]string
	// Specificaly for the README
	CloudPrefix     string
	RepoName        string
	RepoNameEncoded string
}

func GenerateNewBundle(fs afero.Fs, templateData *TemplateData) error {
	return nil
}
