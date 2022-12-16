package templatecache_test

import (
	"fmt"
	"reflect"
	"testing"

	"github.com/massdriver-cloud/mass/internal/templatecache"
	"github.com/spf13/afero"
)

func TestBundleTemplateRefresh(t *testing.T) {
	rootTemplateDir := "/home/md-cloud"
	var fs = afero.NewMemMapFs()

	bundleCache := newMockClient(rootTemplateDir, fs, t)

	err := bundleCache.RefreshTemplates()

	if err != nil {
		t.Error(err)
	}

	got, _ := afero.Glob(fs, fmt.Sprintf("%s/**/*", rootTemplateDir))

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
	var fs = afero.NewMemMapFs()

	directories := []string{
		rootTemplateDir,
		fmt.Sprintf("%s/applications/kubernetes-cronjob", rootTemplateDir),
		fmt.Sprintf("%s/infrastructure/terraform", rootTemplateDir),
		fmt.Sprintf("%s/infrastructure/palumi", rootTemplateDir),
	}

	makeTemplateDirectories(directories, fs, t)

	bundleCache := newMockClient(rootTemplateDir, fs, t)

	got, _ := bundleCache.ListTemplates()

	want := []string{"applications/kubernetes-cronjob", "infrastructure/palumi", "infrastructure/terraform"}

	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, wanted %v", got, want)
	}
}

func TestTemplatePath(t *testing.T) {
	rootTemplateDir := "/home/md-cloud"
	var fs = afero.NewMemMapFs()

	bundleCache := newMockClient(rootTemplateDir, fs, t)

	got, _ := bundleCache.GetTemplatePath()

	if got != rootTemplateDir {
		t.Errorf("got %v, wanted %v", got, rootTemplateDir)
	}
}

func newMockClient(rootTemplateDir string, fs afero.Fs, t *testing.T) templatecache.TemplateCache {
	fetcher := func(filePath string) error {
		directories := []string{
			filePath,
			fmt.Sprintf("%s/applications/aws-lambda", filePath),
			fmt.Sprintf("%s/applications/aws-vm", filePath),
		}

		makeTemplateDirectories(directories, fs, t)

		return nil
	}

	return &templatecache.BundleTemplateCache{
		TemplatePath: rootTemplateDir,
		Fetch:        fetcher,
		Fs:           fs,
	}
}

func makeTemplateDirectories(names []string, fs afero.Fs, t *testing.T) {
	for _, name := range names {
		err := fs.Mkdir(name, 0755)
		if err != nil {
			t.Fatal(err)
		}
	}
}
