package commands_test

import (
	"fmt"
	"reflect"
	"sort"
	"testing"

	"github.com/massdriver-cloud/mass/pkg/commands"
	"github.com/massdriver-cloud/mass/pkg/mockfilesystem"
	"github.com/massdriver-cloud/mass/pkg/templatecache"
	"github.com/spf13/afero"
)

func TestListTemplates(t *testing.T) {
	rootTemplateDir := "/home/md-cloud"
	var fs = afero.NewMemMapFs()

	directories := []string{
		rootTemplateDir,
		fmt.Sprintf("%s/massdriver-cloud/application-templates/kubernetes-cronjob", rootTemplateDir),
		fmt.Sprintf("%s/massdriver-cloud/infrastructure-templates/terraform", rootTemplateDir),
		fmt.Sprintf("%s/massdriver-cloud/infrastructure-templates/palumi", rootTemplateDir),
	}

	err := mockfilesystem.MakeDirectories(directories, fs)

	if err != nil {
		t.Fatal(err)
	}

	files := []mockfilesystem.VirtualFile{
		{Path: fmt.Sprintf("%s/massdriver-cloud/application-templates/kubernetes-cronjob/massdriver.yaml", rootTemplateDir)},
		{Path: fmt.Sprintf("%s/massdriver-cloud/infrastructure-templates/terraform/massdriver.yaml", rootTemplateDir)},
		{Path: fmt.Sprintf("%s/massdriver-cloud/infrastructure-templates/palumi/massdriver.yaml", rootTemplateDir)},
	}

	err = mockfilesystem.MakeFiles(files, fs)

	if err != nil {
		t.Fatal(err)
	}

	bundleCache := templatecache.NewMockClient(rootTemplateDir, fs)

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
