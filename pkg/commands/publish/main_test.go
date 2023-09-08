package publish_test

import (
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/massdriver-cloud/mass/pkg/bundle"
	"github.com/massdriver-cloud/mass/pkg/commands/publish"
	"github.com/massdriver-cloud/mass/pkg/mockfilesystem"
	"github.com/massdriver-cloud/mass/pkg/restclient"
	"github.com/spf13/afero"
)

func TestPublishBundle(t *testing.T) {
	var gotPublishBody []byte

	buildDir := "/publishtest"

	testServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		bytes, err := io.ReadAll(r.Body)
		if err != nil {
			t.Fatalf("%d, unexpected error", err)
		}
		switch r.RequestURI {
		case "/s3":
			break
		case "/bundles":
			responseBody := fmt.Sprintf(`{"upload_location":"http://%s/s3"}`, r.Host)
			gotPublishBody = bytes
			_, err = w.Write([]byte(responseBody))
			if err != nil {
				t.Fatalf("%d, unexpected error writing upload location to test server", err)
			}
		default:
			t.Fatalf("unsupported route %s", r.RequestURI)
		}
	}))
	defer testServer.Close()

	fs := afero.NewMemMapFs()

	err := mockfilesystem.SetupBundle(buildDir, fs)

	if err != nil {
		t.Fatal(err)
	}

	err = mockfilesystem.WithFilesToIgnore(buildDir, fs)

	if err != nil {
		t.Fatal(err)
	}

	err = mockfilesystem.WithOperatorGuide(buildDir, "md", fs)

	if err != nil {
		t.Fatal(err)
	}

	c := restclient.NewClient().WithBaseURL(testServer.URL)
	b := mockBundle()

	err = publish.Run(b, c, fs, buildDir)

	if err != nil {
		t.Fatal(err)
	}

	wantPublishBody := `{"name":"the-bundle","description":"something","type":"bundle","source_url":"github.com/some-repo","access":"public","artifacts_schema":{"artifacts":"foo"},"connections_schema":{"connections":"bar"},"params_schema":{"params":{"hello":"world"}},"ui_schema":{"ui":"baz"},"operator_guide":"IyBTb21lIE1hcmtkb3duIQo=","app":{"envs":{"LOG_LEVEL":"warn"},"policies":[".connections.vpc.data.infrastructure.arn"],"secrets":{"STRIPE_KEY":{"description":"Access key for live stripe accounts","json":false,"required":true,"title":"A secret"}}}}`

	if string(gotPublishBody) != wantPublishBody {
		t.Errorf("expected publish body to be %s but got %s", wantPublishBody, gotPublishBody)
	}
}

func mockBundle() *bundle.Bundle {
	return &bundle.Bundle{
		Name:        "the-bundle",
		Description: "something",
		SourceURL:   "github.com/some-repo",
		Type:        "bundle",
		Access:      "public",
		Steps: []bundle.Step{
			{
				Path:        "deploy",
				Provisioner: "terraform",
			},
		},
		Artifacts: map[string]interface{}{
			"artifacts": "foo",
		},
		Connections: map[string]interface{}{
			"connections": "bar",
		},
		Params: map[string]interface{}{
			"params": map[string]string{
				"hello": "world",
			},
		},
		UI: map[string]interface{}{
			"ui": "baz",
		},
		AppSpec: &bundle.AppSpec{
			Secrets: map[string]bundle.Secret{
				"STRIPE_KEY": {
					Required:    true,
					JSON:        false,
					Title:       "A secret",
					Description: "Access key for live stripe accounts",
				},
			},
			Policies: []string{".connections.vpc.data.infrastructure.arn"},
			Envs: map[string]string{
				"LOG_LEVEL": "warn",
			},
		},
	}
}
