package preview_test

import (
	"reflect"
	"testing"

	"github.com/massdriver-cloud/mass/pkg/api"
	"github.com/massdriver-cloud/mass/pkg/commands/preview"
	"github.com/massdriver-cloud/mass/pkg/gqlmock"
	"github.com/massdriver-cloud/mass/pkg/tui/teahelper"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/massdriver-cloud/massdriver-sdk-go/massdriver/client"
)

func TestRunNew(t *testing.T) {
	projectSlug := "ecomm"

	responses := []any{
		gqlmock.MockQueryResponse("project", map[string]any{
			"slug": projectSlug,
			"defaultParams": map[string]any{
				"database": map[string]any{"username": "root"},
			},
		}),

		gqlmock.MockQueryResponse("artifactDefinitions", []map[string]any{
			{
				"id":    "def-1",
				"name":  "massdriver/aws-iam-role",
				"label": "AWS IAM Role",
				"icon":  "aws",
				"schema": map[string]any{},
				"url":   "https://example.com",
				"ui": map[string]any{
					"connectionOrientation": "VERTICAL",
					"environmentDefaultGroup": "aws",
				},
			},
		}),

		gqlmock.MockQueryResponse("artifacts", map[string]any{
			"next": "",
			"items": []map[string]any{
				{"id": "uuid-here", "name": "aws-credentials"},
			},
		}),
	}

	mdClient := client.Client{
		GQL: gqlmock.NewClientWithJSONResponseArray(responses),
	}

	model, _ := preview.RunNew(t.Context(), &mdClient, projectSlug)

	selectRow := tea.KeyMsg{Type: tea.KeySpace}
	pressNext := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'n'}}

	updatedModel, _ := model.Update(selectRow)
	updatedModel, _ = updatedModel.Update(pressNext)
	updatedModel, _ = updatedModel.Update(selectRow)

	teahelper.AssertModelViewContains(t, updatedModel.View(), "aws-credentials")
	updatedModel, _ = updatedModel.Update(pressNext)

	//nolint:errcheck
	updatedInitializeModel := (updatedModel).(preview.Model)
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
				Params: map[string]any{
					"username": "root",
				},
			},
		},
	}

	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %+v, wanted %+v", got, want)
	}
}
