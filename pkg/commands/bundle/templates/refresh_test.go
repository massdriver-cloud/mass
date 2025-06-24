package templates_test

import (
	"fmt"
	"path"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/massdriver-cloud/mass/pkg/commands/bundle/templates"
	"github.com/massdriver-cloud/mass/pkg/templatecache"
)

func TestRefresh(t *testing.T) {
	testDir := t.TempDir()
	rootTemplateDir := path.Join(testDir, "/home/md-cloud")

	bundleCache := templatecache.NewMockClient(rootTemplateDir)

	err := templates.RunRefresh(bundleCache)

	if err != nil {
		t.Error(err)
	}

	got, _ := filepath.Glob(fmt.Sprintf("%s/**/**/*", rootTemplateDir))

	want := []string{
		path.Join(testDir, "/home/md-cloud/massdriver-cloud/application-templates/aws-lambda"),
		path.Join(testDir, "/home/md-cloud/massdriver-cloud/application-templates/aws-vm"),
	}

	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, wanted %v", got, want)
	}
}
