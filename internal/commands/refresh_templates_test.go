package commands_test

import (
	"fmt"
	"reflect"
	"testing"

	"github.com/massdriver-cloud/mass/internal/commands"
	"github.com/massdriver-cloud/mass/internal/templatecache"
	"github.com/spf13/afero"
)

func TestRefreshTemplates(t *testing.T) {
	rootTemplateDir := "/home/md-cloud"
	var fs = afero.NewMemMapFs()

	bundleCache := templatecache.NewMockClient(rootTemplateDir, fs)

	err := commands.RefreshTemplates(bundleCache)

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
