package templates_test

import (
	"fmt"
	"path"
	"sort"
	"testing"

	"github.com/massdriver-cloud/mass/pkg/commands/bundle/templates"
	"github.com/massdriver-cloud/mass/pkg/mockfilesystem"
	masstemplates "github.com/massdriver-cloud/mass/pkg/templates"
)

func TestList(t *testing.T) {
	testDir := t.TempDir()
	rootTemplateDir := path.Join(testDir, "/home/md-cloud")

	directories := []string{
		rootTemplateDir,
		fmt.Sprintf("%s/kubernetes-cronjob", rootTemplateDir),
		fmt.Sprintf("%s/opentofu", rootTemplateDir),
		fmt.Sprintf("%s/bicep", rootTemplateDir),
	}

	if err := mockfilesystem.MakeDirectories(directories); err != nil {
		t.Fatal(err)
	}

	files := []mockfilesystem.VirtualFile{
		{Path: fmt.Sprintf("%s/kubernetes-cronjob/massdriver.yaml", rootTemplateDir)},
		{Path: fmt.Sprintf("%s/opentofu/massdriver.yaml", rootTemplateDir)},
		{Path: fmt.Sprintf("%s/bicep/massdriver.yaml", rootTemplateDir)},
	}

	if err := mockfilesystem.MakeFiles(files); err != nil {
		t.Fatal(err)
	}

	tmpl := &masstemplates.Templates{Path: rootTemplateDir}

	got, err := templates.RunList(tmpl)
	if err != nil {
		t.Fatal(err)
	}

	want := []string{"bicep", "kubernetes-cronjob", "opentofu"}
	sort.Strings(got)

	if len(got) != len(want) {
		t.Errorf("got %v, wanted %v", got, want)
		return
	}

	for i := range got {
		if got[i] != want[i] {
			t.Errorf("got %v, wanted %v", got, want)
			break
		}
	}
}
