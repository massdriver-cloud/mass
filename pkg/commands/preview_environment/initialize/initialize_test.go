package initialize_test

import (
	"reflect"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/massdriver-cloud/mass/pkg/api"
	"github.com/massdriver-cloud/mass/pkg/commands/preview_environment/initialize"
	"github.com/massdriver-cloud/mass/pkg/gqlmock"
	"github.com/massdriver-cloud/mass/pkg/tui/teahelper"
)

func TestRun(t *testing.T) {
	envSlug := "env1"
	projectSlug := "ecomm1"

	env := map[string]interface{}{
		"name": "potato",
		"slug": "potato",
		"packages": []map[string]interface{}{
			{
				"id": "package1",
				"manifest": map[string]interface{}{
					"slug": "database",
				},
			},
		},
		"project": map[string]interface{}{
			"id": "ecomma",
			"defaultParams": map[string]interface{}{
				"database": map[string]interface{}{
					"username": "root",
				},
			},
		},
	}

	responses := []interface{}{
		gqlmock.MockQueryResponse("environment", env),

		gqlmock.MockQueryResponse("project", map[string]interface{}{
			"slug": projectSlug,
			"defaultParams": map[string]interface{}{
				"database": map[string]interface{}{"username": "root"},
			},
		}),

		gqlmock.MockQueryResponse("artifacts", map[string]interface{}{
			"next": "",
			"items": []map[string]interface{}{
				{"id": "uuid-here", "name": "aws-credentials"},
			},
		}),
	}

	client := gqlmock.NewClientWithJSONResponseArray(responses)

	model, _ := initialize.New(client, "faux-org-id", envSlug)

	selectRow := tea.KeyMsg{Type: tea.KeySpace}
	pressNext := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'n'}}

	updatedModel, _ := model.Update(selectRow)
	updatedModel, _ = updatedModel.Update(pressNext)
	updatedModel, _ = updatedModel.Update(selectRow)

	teahelper.AssertModelViewContains(t, updatedModel.View(), "aws-credentials")
	updatedModel, _ = updatedModel.Update(pressNext)

	//nolint:errcheck
	updatedInitializeModel := (updatedModel).(initialize.Model)
	got := updatedInitializeModel.PreviewConfig()

	want := &api.PreviewConfig{
		ProjectSlug: projectSlug,
		Credentials: []api.Credential{
			{
				ArtifactDefinitionType: "massdriver/aws-iam-role",
				ArtifactId:             "uuid-here",
			},
		},
		Packages: map[string]api.PreviewPackage{
			"database": {
				Params: map[string]interface{}{
					"username": "root",
				},
			},
		},
	}

	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %+v, wanted %+v", got, want)
	}
}
