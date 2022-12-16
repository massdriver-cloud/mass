package template_cache

import (
	"fmt"
	"os"
	"os/user"
	"path/filepath"
	"strings"

	"github.com/go-git/go-git/v5"
	"github.com/spf13/afero"
)

const MassdriverApplicationTemplatesRepository = "https://github.com/massdriver-cloud/application-templates"

type BundleTemplateCache struct {
	TemplatePath string
	Fetch        Fetcher
	Fs           afero.Fs
}

func GithubTemplatesFetcher(writePath string) error {
	_, cloneErr := git.PlainClone(writePath, false, &git.CloneOptions{
		// URL:      common.Config().Application.Templates.Repository,
		URL: MassdriverApplicationTemplatesRepository,
		// Progress: os.Stdout,
		Depth: 1,
	})

	return cloneErr
}

// Refresh available templates from Massdriver official Github repository.
func (b *BundleTemplateCache) RefreshTemplates() error {
	if err := os.RemoveAll(b.TemplatePath); err != nil {
		return err
	}

	fmt.Printf("Downloading templates to %s.\n", b.TemplatePath)
	err := b.Fetch(b.TemplatePath)

	if err != nil {
		return err
	}

	fmt.Printf("Templates added to cache.\n")
	return nil
}

// List all templates available in cache
func (b *BundleTemplateCache) ListTemplates() ([]string, error) {
	matches, err := afero.Glob(b.Fs, fmt.Sprintf("%s/**/*", b.TemplatePath))

	if err != nil {
		return nil, err
	}

	templates := formatTemplateList(matches, b.TemplatePath)

	return templates, nil
}

// Get the path to the template directory
func (b *BundleTemplateCache) GetTemplatePath() (string, error) {
	return b.TemplatePath, nil
}

/*
Template cache factory which will create a new instance of BundleTemplateCache.
Requires a function as a dependency to handle retreival of templates which can in turn be mocked for testing.
*/
func NewBundleTemplateCache(fetch Fetcher, fs afero.Fs) (TemplateCache, error) {
	templatePath, err := getOrCreateTemplateDirectory(fs)

	if err != nil {
		return nil, err
	}

	bundleTemplateCache := &BundleTemplateCache{
		TemplatePath: templatePath,
		Fetch:        fetch,
		Fs:           fs,
	}

	return bundleTemplateCache, nil
}

func getOrCreateTemplateDirectory(fs afero.Fs) (string, error) {
	localDevTemplatesPath := os.Getenv("MD_DEV_TEMPLATES_PATH")

	if localDevTemplatesPath == "" {
		templatesPath, err := doGetOrCreate(fs)
		if err != nil {
			return "", err
		}
		return templatesPath, nil
	} else {
		fmt.Printf("Reading templates for local development path: %s", localDevTemplatesPath)
		return localDevTemplatesPath, nil
	}
}

func doGetOrCreate(fs afero.Fs) (string, error) {
	usr, _ := user.Current()
	dir := usr.HomeDir
	cacheDir := filepath.Join(dir, ".massdriver")
	if _, err := fs.Stat(cacheDir); !os.IsNotExist(err) {
		return cacheDir, err
	}

	if errMkdir := fs.Mkdir(cacheDir, 0755); errMkdir != nil {
		return cacheDir, errMkdir
	}

	return cacheDir, nil
}

func formatTemplateList(templateDirs []string, rootPath string) []string {
	templates := []string{}
	replacement := fmt.Sprintf("%s/", rootPath)
	for _, match := range templateDirs {
		formattedDirectory := strings.Replace(match, replacement, "", 1)
		templates = append(templates, formattedDirectory)
	}

	return templates
}
