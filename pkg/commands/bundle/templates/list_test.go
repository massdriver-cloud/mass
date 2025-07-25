package templates_test

import (
	"fmt"
	"path"
	"reflect"
	"sort"
	"testing"

	"github.com/massdriver-cloud/mass/pkg/commands/bundle/templates"
	"github.com/massdriver-cloud/mass/pkg/mockfilesystem"
	"github.com/massdriver-cloud/mass/pkg/templatecache"
)

func TestList(t *testing.T) {
	testDir := t.TempDir()
	rootTemplateDir := path.Join(testDir, "/home/md-cloud")

	directories := []string{
		rootTemplateDir,
		fmt.Sprintf("%s/massdriver-cloud/application-templates/kubernetes-cronjob", rootTemplateDir),
		fmt.Sprintf("%s/massdriver-cloud/infrastructure-templates/opentofu", rootTemplateDir),
		fmt.Sprintf("%s/massdriver-cloud/infrastructure-templates/bicep", rootTemplateDir),
	}

	err := mockfilesystem.MakeDirectories(directories)

	if err != nil {
		t.Fatal(err)
	}

	files := []mockfilesystem.VirtualFile{
		{Path: fmt.Sprintf("%s/massdriver-cloud/application-templates/kubernetes-cronjob/massdriver.yaml", rootTemplateDir)},
		{Path: fmt.Sprintf("%s/massdriver-cloud/infrastructure-templates/opentofu/massdriver.yaml", rootTemplateDir)},
		{Path: fmt.Sprintf("%s/massdriver-cloud/infrastructure-templates/bicep/massdriver.yaml", rootTemplateDir)},
	}

	err = mockfilesystem.MakeFiles(files)

	if err != nil {
		t.Fatal(err)
	}

	bundleCache := templatecache.NewMockClient(rootTemplateDir)

	got, err := templates.RunList(bundleCache)

	if err != nil {
		t.Fatal(err)
	}

	want := []templatecache.TemplateList{
		{
			Repository: "massdriver-cloud/application-templates",
			Templates:  []string{"kubernetes-cronjob"},
		},
		{
			Repository: "massdriver-cloud/infrastructure-templates",
			Templates:  []string{"bicep", "opentofu"},
		},
	}

	sort.Slice(got, func(i int, j int) bool { return got[i].Repository < got[j].Repository })

	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, wanted %v", got, want)
	}
}
