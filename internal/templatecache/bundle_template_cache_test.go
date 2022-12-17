package templatecache_test

import (
	"fmt"
	"reflect"
	"sort"
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

	got, _ := afero.Glob(fs, fmt.Sprintf("%s/**/**/*", rootTemplateDir))

	want := []string{
		"/home/md-cloud/massdriver-cloud/application-templates/aws-lambda",
		"/home/md-cloud/massdriver-cloud/application-templates/aws-vm",
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
		fmt.Sprintf("%s/massdriver-cloud/application-templates/kubernetes-cronjob", rootTemplateDir),
		fmt.Sprintf("%s/massdriver-cloud/infrastructure-templates/terraform", rootTemplateDir),
		fmt.Sprintf("%s/massdriver-cloud/infrastructure-templates/palumi", rootTemplateDir),
	}

	makeTemplateDirectories(directories, fs, t)

	bundleCache := newMockClient(rootTemplateDir, fs, t)

	got, _ := bundleCache.ListTemplates()

	want := []templatecache.TemplateList{
		{
			Repository: "massdriver-cloud/application-templates",
			Templates:  []string{"kubernetes-cronjob"},
		},
		{
			Repository: "massdriver-cloud/infrastructure-templates",
			Templates:  []string{"palumi", "terraform"},
		},
	}

	sort.Slice(got, func(i int, j int) bool { return got[i].Repository < got[j].Repository })

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
			fmt.Sprintf("%s/massdriver-cloud/application-templates/aws-lambda", filePath),
			fmt.Sprintf("%s/massdriver-cloud/application-templates/aws-vm", filePath),
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

		massdriverConfigFileLocation := fmt.Sprintf("%s/massdriver.yaml", name)
		_, err = fs.Create(massdriverConfigFileLocation)

		if err != nil {
			t.Fatal(err)
		}
	}
}
