package artifacttable_test

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/massdriver-cloud/mass/pkg/api"
	"github.com/massdriver-cloud/mass/pkg/tui/components/artifacttable"
	"github.com/massdriver-cloud/mass/pkg/tui/teahelper"
)

func TestView(t *testing.T) {
	artifacts := []*api.Artifact{
		{Name: "aws iam role", ID: "foobar"},
		{Name: "gcp iam role", ID: "quxqaz"},
	}

	model := artifacttable.New(artifacts)

	teahelper.AssertModelViewContains(t, model.View(), "aws iam role")
	teahelper.AssertModelViewContains(t, model.View(), "quxqaz")
}

func TestUpdateSelectsArtifactDefinition(t *testing.T) {
	awsRole := &api.Artifact{Name: "aws iam role", ID: "foobar"}
	gcpServiceAccount := &api.Artifact{Name: "gcp service account", ID: "quxqaz"}
	artifacts := []*api.Artifact{
		awsRole,
		gcpServiceAccount,
	}

	model := artifacttable.New(artifacts)

	pressDown := tea.KeyMsg{Type: tea.KeyDown}
	updatedModel, _ := model.Update(pressDown)

	pressSpace := tea.KeyMsg{Type: tea.KeySpace}
	updatedModel, _ = updatedModel.Update(pressSpace)

	pressEsc := tea.KeyMsg{Type: tea.KeyEsc}
	updatedModel, _ = updatedModel.Update(pressEsc)

	//nolint:errcheck
	finalModel := (updatedModel).(artifacttable.Model)

	got := finalModel.SelectedArtifacts
	want := gcpServiceAccount

	if len(got) != 1 {
		t.Errorf("Expected exactly one result, got: %v", got)
	}

	if got[0] != want {
		t.Errorf("Got %v, wanted %v", got[0], want)
	}
}
