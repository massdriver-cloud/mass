package resourcetypetable_test

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/massdriver-cloud/mass/internal/api/v1"
	"github.com/massdriver-cloud/mass/internal/tui/components/resourcetypetable"
	"github.com/massdriver-cloud/mass/internal/tui/teahelper"
)

func TestViewHumanizes(t *testing.T) {
	resourceTypes := []*api.ResourceType{
		{Name: "example/password"},
		{Name: "example/iam-thing"},
	}

	model := resourcetypetable.New(resourceTypes)

	teahelper.AssertModelViewContains(t, model.View(), "Password")
	teahelper.AssertModelViewContains(t, model.View(), "IAM Thing")
}

func TestUpdateSelectsResourceType(t *testing.T) {
	want := &api.ResourceType{Name: "example/iam-thing"}
	resourceTypes := []*api.ResourceType{
		{Name: "example/password"},
		want,
	}

	model := resourcetypetable.New(resourceTypes)

	pressDown := tea.KeyMsg{Type: tea.KeyDown}
	updatedModel, _ := model.Update(pressDown)

	pressSpace := tea.KeyMsg{Type: tea.KeySpace}
	updatedModel, _ = updatedModel.Update(pressSpace)

	//nolint:errcheck // type assertion to concrete Model is safe in this test context
	finalModel := (updatedModel).(resourcetypetable.Model)

	got := finalModel.SelectedResourceTypes

	if len(got) != 1 {
		t.Errorf("Expected exactly one result, got: %v", got)
	}

	if got[0] != want {
		t.Errorf("Got %v, wanted %v", got[0], want)
	}
}
