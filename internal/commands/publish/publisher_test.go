package publish_test

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"errors"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"path"
	"reflect"
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
			name:      "Includes the original massdriver.yaml in the request body",
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
				Conf: map[string]interface{}{
					"metadata": map[string]interface{}{
						"foo": "bar",
					},
				},
			},
			wantBody: `{"name":"the-bundle","description":"something","type":"bundle","source_url":"github.com/some-repo","access":"public","artifacts_schema":{"artifacts":"foo"},"connections_schema":{"connections":"bar"},"params_schema":{"params":{"hello":"world"}},"ui_schema":{"ui":"baz"},"conf":{"metadata":{"foo":"bar"}}}`,
		},
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
			},
			wantBody: `{"name":"the-bundle","description":"something","type":"bundle","source_url":"github.com/some-repo","access":"public","artifacts_schema":{"artifacts":"foo"},"connections_schema":{"connections":"bar"},"params_schema":{"params":{"hello":"world"}},"ui_schema":{"ui":"baz"},"operator_guide":"IyBTb21lIE1hcmtkb3duIQo=","app":{"envs":{"LOG_LEVEL":"warn"},"policies":[".connections.vpc.data.infrastructure.arn"],"secrets":{"STRIPE_KEY":{"description":"Access key for live stripe accounts","json":false,"required":true,"title":"A secret"}}}}`,
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
			},
			wantBody: `{"name":"the-bundle","description":"something","type":"bundle","source_url":"github.com/some-repo","access":"public","artifacts_schema":{"artifacts":"foo"},"connections_schema":{"connections":"bar"},"params_schema":{"params":{"hello":"world"}},"ui_schema":{"ui":"baz"},"operator_guide":"IyBTb21lIE1hcmtkb3duIQo=","app":{"envs":{"LOG_LEVEL":"warn"},"policies":[".connections.vpc.data.infrastructure.arn"],"secrets":{"STRIPE_KEY":{"description":"Access key for live stripe accounts","json":false,"required":true,"title":"A secret"}}}}`,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var gotBody string
			testServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				bytes, readErr := io.ReadAll(r.Body)
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
			fs := afero.NewMemMapFs()

			publisher := &publish.Publisher{
				Bundle:     &tc.bundle,
				RestClient: c,
				Fs:         fs,
				BuildDir:   tc.path,
			}

			err := mockfilesystem.SetupBundle(tc.path, fs)

			if err != nil {
				t.Fatal(err)
			}

			err = mockfilesystem.WithOperatorGuide(tc.path, tc.guideType, fs)

			if err != nil {
				t.Fatal(err)
			}

			gotResponse, err := publisher.SubmitBundle()

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

func TestArchive(t *testing.T) {
	b := bundle.Bundle{
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

	fs := afero.NewMemMapFs()
	buildDir := "/archive"

	err := mockfilesystem.SetupBundle(buildDir, fs)

	if err != nil {
		t.Fatal(err)
	}

	err = mockfilesystem.WithFilesToIgnore(buildDir, fs)

	if err != nil {
		t.Fatal(err)
	}

	publisher := &publish.Publisher{
		Bundle:   &b,
		Fs:       fs,
		BuildDir: buildDir,
	}

	var buf bytes.Buffer

	err = publisher.ArchiveBundle(&buf)

	if err != nil {
		t.Fatal(err)
	}

	extractTarGz(bytes.NewReader(buf.Bytes()), fs)

	wantSrc := []string{"main.tf"}
	wantTopLevel := []string{"deploy", "massdriver.yaml", "src"}

	assertDirContains(wantTopLevel, "/untar/bundle", fs, t)
	assertDirContains(wantSrc, "/untar/bundle/src", fs, t)
}

func TestUploadToPresignedS3URL(t *testing.T) {
	type test struct {
		name  string
		bytes []byte
	}
	tests := []test{
		{
			name:  "simple",
			bytes: []byte{1, 2, 3, 4},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var gotBody []byte
			testServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				bytes, err := io.ReadAll(r.Body)
				if err != nil {
					t.Fatalf("%d, unexpected error", err)
				}
				gotBody = bytes

				w.WriteHeader(http.StatusOK)
			}))
			defer testServer.Close()

			publisher := publish.Publisher{}

			err := publisher.PushArchiveToPackageManager(testServer.URL, bytes.NewReader(tc.bytes))
			if err != nil {
				t.Fatalf("%d, unexpected error", err)
			}

			if string(gotBody) != string(tc.bytes) {
				t.Errorf("got %v, want %v", gotBody, tc.bytes)
			}
		})
	}
}

func assertDirContains(want []string, dir string, fs afero.Fs, t *testing.T) {
	got := []string{}

	info, err := afero.ReadDir(fs, dir)

	if err != nil {
		t.Fatal(err)
	}

	for _, val := range info {
		got = append(got, val.Name())
	}

	if !reflect.DeepEqual(want, got) {
		t.Errorf("Expected %v got %v", want, got)
	}
}

func extractTarGz(gzipStream io.Reader, fs afero.Fs) {
	uncompressedStream, err := gzip.NewReader(gzipStream)
	if err != nil {
		log.Fatal("ExtractTarGz: NewReader failed")
	}

	tarReader := tar.NewReader(uncompressedStream)

	for {
		header, tarErr := tarReader.Next()

		if errors.Is(tarErr, io.EOF) {
			break
		}

		if tarErr != nil {
			log.Fatalf("ExtractTarGz: Next() failed: %s", err.Error())
		}

		switch header.Typeflag {
		case tar.TypeDir:
			dirErr := fs.Mkdir(path.Join("/untar/", header.Name), 0755)
			if dirErr != nil {
				log.Fatalf("ExtractTarGz: Mkdir() failed: %s", err.Error())
			}
		case tar.TypeReg:
			outFile, headerErr := fs.Create(path.Join("/untar/", header.Name))
			if headerErr != nil {
				log.Fatalf("ExtractTarGz: Create() failed: %s", err.Error())
			}

			_, err = io.Copy(outFile, tarReader)

			if err != nil {
				outFile.Close()
				log.Fatalf("ExtractTarGz: Copy() failed: %s", err.Error())
			}

			outFile.Close()
		default:
			log.Fatalf(
				"ExtractTarGz: uknown type: %v in %s",
				header.Typeflag,
				header.Name)
		}
	}
}
