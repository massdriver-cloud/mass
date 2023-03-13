package templatecache

import (
	"fmt"
	"os"
	"os/user"
	"path"
	"path/filepath"
	"strings"

	"github.com/spf13/afero"
)

/*
Adding additional repositories will create the appropriate subdirectories and list
them accordingly in bunde templates list. In the future this should be read form .massrc.
*/
var massdriverApplicationTemplatesRepositories = []string{
	"https://github.com/massdriver-cloud/application-templates",
}

const gitOrg = "*"
const repoName = "*"
const templateDir = "*"

type BundleTemplateCache struct {
	TemplatePath string
	Fetch        Fetcher
	Fs           afero.Fs
}

type TemplateList struct {
	Repository string
	Templates  []string
}

type CloneError struct {
	Repository string
	Error      string
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
func (b *BundleTemplateCache) ListTemplates() ([]TemplateList, error) {
	/*
		Go does not support ** glob matching: https://github.com/golang/go/issues/11862
		If we want to support arbitrarily nested matching we will likely have to introduce this library: https://github.com/bmatcuk/doublestar
		Issue here: https://linear.app/massdriver/issue/PLAT-262/support-glob-in-mass-cli-for-arbitrarily-nested-template-repositories
	*/
	matches, err := afero.Glob(b.Fs, fmt.Sprintf("%s/%s/%s/%s/massdriver.yaml", b.TemplatePath, gitOrg, repoName, templateDir))

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
Clones the desired template to the directory specified by the user and renders the massdriver YAML with user supplied values.
*/
func (b *BundleTemplateCache) RenderTemplate(data *TemplateData) error {
	fileManager := &fileManager{
		fs:                    b.Fs,
		readDirectory:         path.Join(data.TemplateSource, data.TemplateRepo, data.TemplateName),
		writeDirectory:        data.OutputDir,
		templateData:          data,
		templateRootDirectory: b.TemplatePath,
	}

	err := fileManager.CopyTemplate()

	if err != nil {
		return err
	}

	return nil
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
	localDevTemplatesPath := os.Getenv("MD_TEMPLATES_PATH")

	if localDevTemplatesPath == "" {
		templatesPath, err := doGetOrCreate(fs)
		if err != nil {
			return "", err
		}
		return templatesPath, nil
	}

	return localDevTemplatesPath, nil
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

func formatTemplateList(templateDirs []string, rootPath string) []TemplateList {
	templatesMap := make(map[string][]string)
	replacement := fmt.Sprintf("%s/", rootPath)
	for _, match := range templateDirs {
		formattedDirectory := strings.Replace(match, replacement, "", 1)
		pathParts := strings.Split(formattedDirectory, "/")
		repository := fmt.Sprintf("%s/%s", pathParts[0], pathParts[1])
		templatesMap[repository] = append(templatesMap[repository], pathParts[2])
	}

	templateList := buildTemplateList(templatesMap)

	return templateList
}

func buildTemplateList(templateMap map[string][]string) []TemplateList {
	templateList := []TemplateList{}

	for k, v := range templateMap {
		templateList = append(templateList, TemplateList{Repository: k, Templates: v})
	}

	return templateList
}
