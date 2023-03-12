package commands_test

import (
	"bytes"
	"reflect"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/massdriver-cloud/mass/internal/commands"
	"github.com/massdriver-cloud/mass/internal/gqlmock"
	"github.com/massdriver-cloud/mass/internal/tui/teahelper"
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

		gqlmock.MockQueryResponse("artifacts", map[string]interface{}{
			"next": "foo",
			"items": []map[string]interface{}{
				{"id": "uuid-here", "name": "aws-credentials"},
			},
		}),
	}

	client := gqlmock.NewClientWithJSONResponseArray(responses)
	var stdin bytes.Buffer
	var stdout bytes.Buffer

	go func() {
		// Select type
		time.Sleep(10 * time.Millisecond)
		teahelper.SendSpecialKeyPress(commands.P, tea.KeyEnter)
		teahelper.SendKeyPresses(commands.P, "s")

		// Select credential
		time.Sleep(10 * time.Millisecond)
		teahelper.SendSpecialKeyPress(commands.P, tea.KeyEnter)
		teahelper.SendKeyPresses(commands.P, "s")
	}()

	previewCfg, err := commands.InitializePreviewEnvironment(client, "faux-org-id", projectSlug, &stdin, &stdout)

	teahelper.AssertUIContains(t, stdout, "aws-iam-role")

	if err != nil {
		t.Fatal(err)
	}

	got := previewCfg
	want := &commands.PreviewConfig{
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
