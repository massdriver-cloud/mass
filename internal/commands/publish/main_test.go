package publish_test

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"

	"github.com/massdriver-cloud/mass/internal/bundle"
	"github.com/massdriver-cloud/mass/internal/commands/publish"
	"github.com/massdriver-cloud/mass/internal/mockfilesystem"
	"github.com/massdriver-cloud/mass/internal/restclient"
	"github.com/spf13/afero"
)

func TestPublishBundle(t *testing.T) {
	var gotPublishBody []byte

	buildDir := "/publishtest"

	testServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		bytes, err := ioutil.ReadAll(r.Body)
		if err != nil {
			t.Fatalf("%d, unexpected error", err)
		}
		switch r.RequestURI {
		case "/s3":
			break
		case "/bundles":
			responseBody := fmt.Sprintf(`{"upload_location":"http://%s/s3"}`, r.Host)
			gotPublishBody = bytes
			if _, err := w.Write([]byte(responseBody)); err != nil {
				t.Fatalf("%d, unexpected error writing upload location to test server", err)
			}
		default:
			t.Fatalf("unsupported route %s", r.RequestURI)
		}
	}))
	defer testServer.Close()

	fs := afero.NewMemMapFs()

	mockfilesystem.SetupBundle(buildDir, fs)
	mockfilesystem.WithFilesToIgnore(buildDir, fs)
	mockfilesystem.WithOperatorGuide(buildDir, "md", fs)

	c := restclient.NewClient().WithBaseURL(testServer.URL)
	b := mockBundle()

	err := publish.Run(b, c, fs, buildDir)

	if err != nil {
		t.Fatal(err)
	}

	wantPublishBody := []byte{123, 34, 110, 97, 109, 101, 34, 58, 34, 116, 104, 101, 45, 98, 117, 110, 100, 108, 101, 34, 44, 34, 100, 101, 115, 99, 114, 105, 112, 116, 105, 111, 110, 34, 58, 34, 115, 111, 109, 101, 116, 104, 105, 110, 103, 34, 44, 34, 116, 121, 112, 101, 34, 58, 34, 98, 117, 110, 100, 108, 101, 34, 44, 34, 115, 111, 117, 114, 99, 101, 95, 117, 114, 108, 34, 58, 34, 103, 105, 116, 104, 117, 98, 46, 99, 111, 109, 47, 115, 111, 109, 101, 45, 114, 101, 112, 111, 34, 44, 34, 97, 99, 99, 101, 115, 115, 34, 58, 34, 112, 117, 98, 108, 105, 99, 34, 44, 34, 97, 114, 116, 105, 102, 97, 99, 116, 115, 95, 115, 99, 104, 101, 109, 97, 34, 58, 123, 34, 97, 114, 116, 105, 102, 97, 99, 116, 115, 34, 58, 34, 102, 111, 111, 34, 125, 44, 34, 99, 111, 110, 110, 101, 99, 116, 105, 111, 110, 115, 95, 115, 99, 104, 101, 109, 97, 34, 58, 123, 34, 99, 111, 110, 110, 101, 99, 116, 105, 111, 110, 115, 34, 58, 34, 98, 97, 114, 34, 125, 44, 34, 112, 97, 114, 97, 109, 115, 95, 115, 99, 104, 101, 109, 97, 34, 58, 123, 34, 112, 97, 114, 97, 109, 115, 34, 58, 123, 34, 104, 101, 108, 108, 111, 34, 58, 34, 119, 111, 114, 108, 100, 34, 125, 125, 44, 34, 117, 105, 95, 115, 99, 104, 101, 109, 97, 34, 58, 123, 34, 117, 105, 34, 58, 34, 98, 97, 122, 34, 125, 44, 34, 111, 112, 101, 114, 97, 116, 111, 114, 95, 103, 117, 105, 100, 101, 34, 58, 34, 73, 121, 66, 84, 98, 50, 49, 108, 73, 69, 49, 104, 99, 109, 116, 107, 98, 51, 100, 117, 73, 81, 61, 61, 34, 44, 34, 97, 112, 112, 34, 58, 123, 34, 101, 110, 118, 115, 34, 58, 123, 34, 76, 79, 71, 95, 76, 69, 86, 69, 76, 34, 58, 34, 119, 97, 114, 110, 34, 125, 44, 34, 112, 111, 108, 105, 99, 105, 101, 115, 34, 58, 91, 34, 46, 99, 111, 110, 110, 101, 99, 116, 105, 111, 110, 115, 46, 118, 112, 99, 46, 100, 97, 116, 97, 46, 105, 110, 102, 114, 97, 115, 116, 114, 117, 99, 116, 117, 114, 101, 46, 97, 114, 110, 34, 93, 44, 34, 115, 101, 99, 114, 101, 116, 115, 34, 58, 123, 34, 83, 84, 82, 73, 80, 69, 95, 75, 69, 89, 34, 58, 123, 34, 100, 101, 115, 99, 114, 105, 112, 116, 105, 111, 110, 34, 58, 34, 65, 99, 99, 101, 115, 115, 32, 107, 101, 121, 32, 102, 111, 114, 32, 108, 105, 118, 101, 32, 115, 116, 114, 105, 112, 101, 32, 97, 99, 99, 111, 117, 110, 116, 115, 34, 44, 34, 106, 115, 111, 110, 34, 58, 102, 97, 108, 115, 101, 44, 34, 114, 101, 113, 117, 105, 114, 101, 100, 34, 58, 116, 114, 117, 101, 44, 34, 116, 105, 116, 108, 101, 34, 58, 34, 65, 32, 115, 101, 99, 114, 101, 116, 34, 125, 125, 125, 125}

	if !reflect.DeepEqual(gotPublishBody, wantPublishBody) {
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
	}
}
