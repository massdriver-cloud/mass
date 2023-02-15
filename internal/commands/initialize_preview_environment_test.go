package commands_test

import (
	"reflect"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/massdriver-cloud/mass/internal/commands"
	"github.com/massdriver-cloud/mass/internal/gqlmock"
)

func TestInitializePreviewEnvironment(t *testing.T) {
	projectSlug := "ecomm"

	responses := []interface{}{
		gqlmock.MockQueryResponse("project", map[string]interface{}{
			"slug": projectSlug,
			"defaultParams": map[string]interface{}{
				"database": map[string]interface{}{"username": "root"},
			},
		}),
	}

	client := gqlmock.NewClientWithJSONResponseArray(responses)
	previewCfg, err := commands.InitializePreviewEnvironment(client, "faux-org-id", projectSlug)

	if err != nil {
		t.Fatal(err)
	}

	got := previewCfg.PackageParams
	want := map[string]interface{}{
		"database": map[string]interface{}{
			"username": "root",
		},
	}

	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, wanted %v", got, want)
	}
}

func keyPress(key rune) tea.Msg {
	return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{key}, Alt: false}
}
