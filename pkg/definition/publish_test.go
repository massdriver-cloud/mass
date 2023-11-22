package definition_test

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/massdriver-cloud/mass/pkg/definition"
	"github.com/massdriver-cloud/mass/pkg/restclient"
)

func TestPublish(t *testing.T) {
	type test struct {
		name       string
		definition *bytes.Buffer
		wantBody   string
	}
	tests := []test{
		{
			name:       "simple",
			definition: bytes.NewBuffer([]byte(`{"$md":{"access":"public","name":"foo"},"required":["data","specs"],"properties":{"data":{},"specs":{}}}`)),
			wantBody:   `{"$md":{"access":"public","name":"foo"},"required":["data","specs"],"properties":{"data":{},"specs":{}}}`,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var gotBody string
			testServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				bytes, err := io.ReadAll(r.Body)
				if err != nil {
					t.Fatalf("%d, unexpected error", err)
				}
				gotBody = string(bytes)
				w.WriteHeader(http.StatusOK)
			}))
			defer testServer.Close()

			c := restclient.NewClient().WithBaseURL(testServer.URL)

			err := definition.Publish(c, tc.definition)
			if err != nil {
				t.Fatalf("%d, unexpected error", err)
			}

			if gotBody != tc.wantBody {
				t.Errorf("got %v, want %v", gotBody, tc.wantBody)
			}
		})
	}
}
