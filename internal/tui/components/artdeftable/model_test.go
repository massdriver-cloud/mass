package artdeftable_test

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/massdriver-cloud/mass/internal/api"
	"github.com/massdriver-cloud/mass/internal/tui/components/artdeftable"
	"github.com/massdriver-cloud/mass/internal/tui/teahelper"
)

func TestViewHumanizes(t *testing.T) {
	artdefs := []*api.ArtifactDefinition{
		{Name: "example/password"},
		{Name: "example/iam-thing"},
	}

	model := artdeftable.New(artdefs)

	teahelper.AssertModelViewContains(t, model.View(), "Password")
	teahelper.AssertModelViewContains(t, model.View(), "IAM Thing")
}

func TestUpdateSelectsArtifactDefinition(t *testing.T) {
	want := &api.ArtifactDefinition{Name: "example/iam-thing"}
	artdefs := []*api.ArtifactDefinition{
		{Name: "example/password"},
		want,
	}

	model := artdeftable.New(artdefs)

	pressDown := tea.KeyMsg{Type: tea.KeyDown}
	updatedModel, _ := model.Update(pressDown)

	pressSpace := tea.KeyMsg{Type: tea.KeySpace}
	updatedModel, _ = updatedModel.Update(pressSpace)

	finalModel := (updatedModel).(artdeftable.Model)

	got := finalModel.SelectedArtifactDefinitions

	if len(got) != 1 {
		t.Errorf("Expected exactly one result, got: %v", got)
	}

	if got[0] != want {
		t.Errorf("Got %v, wanted %v", got[0], want)
	}
}
