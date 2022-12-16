package template_cache_test

import (
	"fmt"
	"reflect"
	"testing"

	"github.com/massdriver-cloud/mass/internal/template_cache"
	"github.com/spf13/afero"
)

func TestBundleTemplateRefresh(t *testing.T) {
	rootTemplateDir := "/home/md-cloud"
	var fs afero.Fs = afero.NewMemMapFs()
	fs.Mkdir(rootTemplateDir, 0755)

	bundleCache := newMockClient(rootTemplateDir, fs)

	bundleCache.RefreshTemplates()
	matches, _ := afero.Glob(fs, fmt.Sprintf("%s/**/*", rootTemplateDir))

	got := []string{}

	for _, match := range matches {
		got = append(got, match)
	}

	want := []string{
		fmt.Sprintf("%s/applications/aws-lambda", rootTemplateDir),
		fmt.Sprintf("%s/applications/aws-vm", rootTemplateDir),
	}

	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, wanted %v", got, want)
	}
}

func TestListTemplates(t *testing.T) {
	rootTemplateDir := "/home/md-cloud"
	var fs afero.Fs = afero.NewMemMapFs()
	fs.Mkdir(rootTemplateDir, 0755)
	fs.Mkdir(fmt.Sprintf("%s/applications/kubernetes-cronjob", rootTemplateDir), 0755)
	fs.Mkdir(fmt.Sprintf("%s/infrastructure/terraform", rootTemplateDir), 0755)
	fs.Mkdir(fmt.Sprintf("%s/infrastructure/palumi", rootTemplateDir), 0755)

	bundleCache := newMockClient(rootTemplateDir, fs)

	got, _ := bundleCache.ListTemplates()

	want := []string{"applications/kubernetes-cronjob", "infrastructure/palumi", "infrastructure/terraform"}

	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, wanted %v", got, want)
	}
}

func TestTemplatePath(t *testing.T) {
	rootTemplateDir := "/home/md-cloud"
	var fs afero.Fs = afero.NewMemMapFs()

	bundleCache := newMockClient(rootTemplateDir, fs)

	got, _ := bundleCache.GetTemplatePath()

	if got != rootTemplateDir {
		t.Errorf("got %v, wanted %v", got, rootTemplateDir)
	}
}

func newMockClient(rootTemplateDir string, fs afero.Fs) template_cache.TemplateCache {
	fetcher := func(filePath string) error {
		fs.Mkdir(filePath, 0755)
		fs.Mkdir(fmt.Sprintf("%s/applications/aws-lambda", filePath), 0755)
		fs.Mkdir(fmt.Sprintf("%s/applications/aws-vm", filePath), 0755)

		return nil
	}

	return &template_cache.BundleTemplateCache{
		TemplatePath: rootTemplateDir,
		Fetch:        fetcher,
		Fs:           fs,
	}
}
