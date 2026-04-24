package resourcetable_test

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/massdriver-cloud/mass/internal/api/v1"
	"github.com/massdriver-cloud/mass/internal/tui/components/resourcetable"
	"github.com/massdriver-cloud/mass/internal/tui/teahelper"
)

func TestView(t *testing.T) {
	resources := []*api.Resource{
		{Name: "aws iam role", ID: "foobar"},
		{Name: "gcp iam role", ID: "quxqaz"},
	}

	model := resourcetable.New(resources)

	teahelper.AssertModelViewContains(t, model.View(), "aws iam role")
	teahelper.AssertModelViewContains(t, model.View(), "quxqaz")
}

func TestUpdateSelectsResource(t *testing.T) {
	awsRole := &api.Resource{Name: "aws iam role", ID: "foobar"}
	gcpServiceAccount := &api.Resource{Name: "gcp service account", ID: "quxqaz"}
	resources := []*api.Resource{
		awsRole,
		gcpServiceAccount,
	}

	model := resourcetable.New(resources)

	pressDown := tea.KeyMsg{Type: tea.KeyDown}
	updatedModel, _ := model.Update(pressDown)

	pressSpace := tea.KeyMsg{Type: tea.KeySpace}
	updatedModel, _ = updatedModel.Update(pressSpace)

	pressEsc := tea.KeyMsg{Type: tea.KeyEsc}
	updatedModel, _ = updatedModel.Update(pressEsc)

	//nolint:errcheck // type assertion to concrete Model is safe in this test context
	finalModel := (updatedModel).(resourcetable.Model)

	got := finalModel.SelectedResources
	want := gcpServiceAccount

	if len(got) != 1 {
		t.Errorf("Expected exactly one result, got: %v", got)
	}

	if got[0] != want {
		t.Errorf("Got %v, wanted %v", got[0], want)
	}
}
