package definition_test

import (
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/massdriver-cloud/mass/internal/definition"
	"github.com/massdriver-cloud/mass/internal/restclient"
)

func TestPublish(t *testing.T) {
	type test struct {
		name       string
		definition *definition.Definition
		wantBody   string
	}
	tests := []test{
		{
			name: "simple",
			definition: &definition.Definition{
				"definition": map[string]interface{}{
					"foo": "bar",
				},
			},
			wantBody: `{"definition":{"foo":"bar"}}`,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var gotBody string
			testServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				bytes, err := ioutil.ReadAll(r.Body)
				if err != nil {
					t.Fatalf("%d, unexpected error", err)
				}
				gotBody = string(bytes)
				w.WriteHeader(http.StatusOK)
			}))
			defer testServer.Close()

			c := restclient.NewClient().WithBaseURL(testServer.URL)

			err := tc.definition.Publish(c)
			if err != nil {
				t.Fatalf("%d, unexpected error", err)
			}

			if gotBody != tc.wantBody {
				t.Errorf("got %v, want %v", gotBody, tc.wantBody)
			}
		})
	}
}
