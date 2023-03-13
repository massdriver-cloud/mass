package initialize_test

import (
	"reflect"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/massdriver-cloud/mass/internal/api"
	"github.com/massdriver-cloud/mass/internal/commands/preview_environments/initialize"
	"github.com/massdriver-cloud/mass/internal/gqlmock"
	"github.com/massdriver-cloud/mass/internal/tui/teahelper"
)

func TestRun(t *testing.T) {
	projectSlug := "ecomm"

	responses := []interface{}{
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

	model, _ := initialize.Run(client, "faux-org-id", projectSlug)

	selectRow := tea.KeyMsg{Type: tea.KeySpace}
	pressNext := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'n'}}

	updatedModel, _ := model.Update(selectRow)
	updatedModel, _ = updatedModel.Update(pressNext)
	updatedModel, _ = updatedModel.Update(selectRow)

	teahelper.AssertModelViewContains(t, updatedModel.View(), "aws-credentials")
	updatedModel, _ = updatedModel.Update(pressNext)

	updatedInitializeModel := (updatedModel).(initialize.Model)
	got := updatedInitializeModel.PreviewConfig()

	want := &api.PreviewConfig{
		PackageParams: map[string]interface{}{
			"database": map[string]interface{}{
				"username": "root",
			},
		},
		Credentials: map[string]string{
			"massdriver/aws-iam-role": "uuid-here",
		},
	}

	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, wanted %v", got, want)
	}
}
