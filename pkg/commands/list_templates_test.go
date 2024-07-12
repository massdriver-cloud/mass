package commands_test

import (
	"fmt"
	"path"
	"reflect"
	"sort"
	"testing"

	"github.com/massdriver-cloud/mass/pkg/commands"
	"github.com/massdriver-cloud/mass/pkg/mockfilesystem"
	"github.com/massdriver-cloud/mass/pkg/templatecache"
)

func TestListTemplates(t *testing.T) {
	testDir := t.TempDir()
	rootTemplateDir := path.Join(testDir, "/home/md-cloud")

	directories := []string{
		rootTemplateDir,
		fmt.Sprintf("%s/massdriver-cloud/application-templates/kubernetes-cronjob", rootTemplateDir),
		fmt.Sprintf("%s/massdriver-cloud/infrastructure-templates/terraform", rootTemplateDir),
		fmt.Sprintf("%s/massdriver-cloud/infrastructure-templates/palumi", rootTemplateDir),
	}

	err := mockfilesystem.MakeDirectories(directories)

	if err != nil {
		t.Fatal(err)
	}

	files := []mockfilesystem.VirtualFile{
		{Path: fmt.Sprintf("%s/massdriver-cloud/application-templates/kubernetes-cronjob/massdriver.yaml", rootTemplateDir)},
		{Path: fmt.Sprintf("%s/massdriver-cloud/infrastructure-templates/terraform/massdriver.yaml", rootTemplateDir)},
		{Path: fmt.Sprintf("%s/massdriver-cloud/infrastructure-templates/palumi/massdriver.yaml", rootTemplateDir)},
	}

	err = mockfilesystem.MakeFiles(files)

	if err != nil {
		t.Fatal(err)
	}

	bundleCache := templatecache.NewMockClient(rootTemplateDir)

	got, err := commands.ListTemplates(bundleCache)

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
			Templates:  []string{"palumi", "terraform"},
		},
	}

	sort.Slice(got, func(i int, j int) bool { return got[i].Repository < got[j].Repository })

	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, wanted %v", got, want)
	}
}
