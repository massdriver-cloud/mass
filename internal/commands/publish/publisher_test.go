package publish_test

import (
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/massdriver-cloud/mass/internal/bundle"
	"github.com/massdriver-cloud/mass/internal/commands/publish"
	"github.com/massdriver-cloud/mass/internal/mockfilesystem"
	"github.com/massdriver-cloud/mass/internal/restclient"
	"github.com/spf13/afero"
)

func TestPublish(t *testing.T) {
	type test struct {
		name      string
		path      string
		guideType string
		bundle    bundle.Bundle
		wantBody  string
	}
	tests := []test{
		{
			name:      "Does not submit an app block field if one does not exist",
			path:      "./templates",
			guideType: "",
			bundle: bundle.Bundle{
				Name:        "the-bundle",
				Description: "something",
				SourceURL:   "github.com/some-repo",
				Type:        "bundle",
				Access:      "public",
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
				AppSpec: nil,
			},
			wantBody: `{"name":"the-bundle","description":"something","type":"bundle","source_url":"github.com/some-repo","access":"public","artifacts_schema":{"artifacts":"foo"},"connections_schema":{"connections":"bar"},"params_schema":{"params":{"hello":"world"}},"ui_schema":{"ui":"baz"}}`,
		},
		{
			name:      "Submits an app block field if one does exist",
			path:      "./templates",
			guideType: "",
			bundle: bundle.Bundle{
				Name:        "the-bundle",
				Description: "something",
				SourceURL:   "github.com/some-repo",
				Type:        "bundle",
				Access:      "public",
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
							Json:        false,
							Title:       "A secret",
							Description: "Access key for live stripe accounts",
						},
					},
					Policies: []string{".connections.vpc.data.infrastructure.arn"},
					Envs: map[string]string{
						"LOG_LEVEL": "warn",
					},
				},
			},
			wantBody: `{"name":"the-bundle","description":"something","type":"bundle","source_url":"github.com/some-repo","access":"public","artifacts_schema":{"artifacts":"foo"},"connections_schema":{"connections":"bar"},"params_schema":{"params":{"hello":"world"}},"ui_schema":{"ui":"baz"},"app":{"envs":{"LOG_LEVEL":"warn"},"policies":[".connections.vpc.data.infrastructure.arn"],"secrets":{"STRIPE_KEY":{"description":"Access key for live stripe accounts","json":false,"required":true,"title":"A secret"}}}}`,
		},
		{
			name:      "Submits an operator.md guide if it exist",
			path:      "/md",
			guideType: "md",
			bundle: bundle.Bundle{
				Name:        "the-bundle",
				Description: "something",
				SourceURL:   "github.com/some-repo",
				Type:        "bundle",
				Access:      "public",
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
							Json:        false,
							Title:       "A secret",
							Description: "Access key for live stripe accounts",
						},
					},
					Policies: []string{".connections.vpc.data.infrastructure.arn"},
					Envs: map[string]string{
						"LOG_LEVEL": "warn",
					},
				},
			},
			wantBody: `{"name":"the-bundle","description":"something","type":"bundle","source_url":"github.com/some-repo","access":"public","artifacts_schema":{"artifacts":"foo"},"connections_schema":{"connections":"bar"},"params_schema":{"params":{"hello":"world"}},"ui_schema":{"ui":"baz"},"operator_guide":"IyBTb21lIE1hcmtkb3duIQ==","app":{"envs":{"LOG_LEVEL":"warn"},"policies":[".connections.vpc.data.infrastructure.arn"],"secrets":{"STRIPE_KEY":{"description":"Access key for live stripe accounts","json":false,"required":true,"title":"A secret"}}}}`,
		},
		{
			name:      "Submits an operator.mdx guide if it exist",
			path:      "/mdx",
			guideType: "mdx",
			bundle: bundle.Bundle{
				Name:        "the-bundle",
				Description: "something",
				SourceURL:   "github.com/some-repo",
				Type:        "bundle",
				Access:      "public",
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
							Json:        false,
							Title:       "A secret",
							Description: "Access key for live stripe accounts",
						},
					},
					Policies: []string{".connections.vpc.data.infrastructure.arn"},
					Envs: map[string]string{
						"LOG_LEVEL": "warn",
					},
				},
			},
			wantBody: `{"name":"the-bundle","description":"something","type":"bundle","source_url":"github.com/some-repo","access":"public","artifacts_schema":{"artifacts":"foo"},"connections_schema":{"connections":"bar"},"params_schema":{"params":{"hello":"world"}},"ui_schema":{"ui":"baz"},"operator_guide":"IyBTb21lIE1hcmtkb3duIQ==","app":{"envs":{"LOG_LEVEL":"warn"},"policies":[".connections.vpc.data.infrastructure.arn"],"secrets":{"STRIPE_KEY":{"description":"Access key for live stripe accounts","json":false,"required":true,"title":"A secret"}}}}`,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var gotBody string
			testServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				bytes, readErr := ioutil.ReadAll(r.Body)
				if readErr != nil {
					t.Fatalf("%d, unexpected error", readErr)
				}
				gotBody = string(bytes)

				if _, err := w.Write([]byte(`{"upload_location":"https://some.site.test/endpoint"}`)); err != nil {
					t.Fatalf("%d, unexpected error writing upload location to test server", err)
				}
				w.WriteHeader(http.StatusOK)
			}))
			defer testServer.Close()

			c := restclient.NewClient().WithBaseURL(testServer.URL)

			publisher := &publish.Publisher{
				Bundle:     &tc.bundle,
				RestClient: *c,
			}

			fs := afero.NewMemMapFs()

			mockfilesystem.SetupBundle(tc.path, fs)
			mockfilesystem.WithOperatorGuide(tc.path, tc.guideType, fs)

			gotResponse, err := publisher.SubmitBundle(tc.path, fs)

			if err != nil {
				t.Fatalf("%d, unexpected error", err)
			}

			if gotBody != tc.wantBody {
				t.Errorf("got %v, want %v", gotBody, tc.wantBody)
			}
			if gotResponse != `https://some.site.test/endpoint` {
				t.Errorf("got %v, want %v", gotResponse, `https://some.site.test/endpoint`)
			}
		})
	}
}
