package definition_test

import (
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"

	"github.com/massdriver-cloud/mass/internal/definition"
	"github.com/massdriver-cloud/mass/internal/restclient"
)

func TestGet(t *testing.T) {
	type test struct {
		name       string
		definition string
		want       map[string]interface{}
	}
	tests := []test{
		{
			name:       "simple",
			definition: `{"foo":{"hello":"world"}}`,
			want: map[string]interface{}{
				"foo": map[string]interface{}{
					"hello": "world",
				},
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			testServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				urlPath := r.URL.Path
				switch urlPath {
				case "/artifact-definitions/massdriver/test-schema":
					if _, err := w.Write([]byte(tc.definition)); err != nil {
						t.Errorf("Encountered error writing schema: %v", err)
					}
				default:
					t.Fatalf("unknown schema: %v", urlPath)
				}
			}))
			defer testServer.Close()

			c := restclient.NewClient().WithBaseURL(testServer.URL)

			got, err := definition.GetDefinition(c, "massdriver/test-schema")
			if err != nil {
				t.Fatalf("%d, unexpected error", err)
			}

			if !reflect.DeepEqual(got, tc.want) {
				t.Errorf("got %v, want %v", got, tc.want)
			}
		})
	}
}
