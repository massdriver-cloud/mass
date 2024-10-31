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
	"os"
	"path"
	"reflect"
	"testing"

	"github.com/massdriver-cloud/mass/pkg/bundle"
	"github.com/massdriver-cloud/mass/pkg/commands/publish"
	"github.com/massdriver-cloud/mass/pkg/mockfilesystem"
	"github.com/massdriver-cloud/mass/pkg/restclient"
)

func TestPublish(t *testing.T) {
	type test struct {
		name      string
		guideType string
		bundle    bundle.Bundle
		wantBody  string
	}
	tests := []test{
		{
			name:      "Does not submit an app block field if one does not exist",
			guideType: "",
			bundle: bundle.Bundle{
				Name:        "the-bundle",
				Description: "something",
				SourceURL:   "github.com/some-repo",
				Type:        "bundle",
				Access:      "public",
				Steps: []bundle.Step{
					{
						Path:        "deploy",
						Provisioner: "opentofu",
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
				AppSpec: nil,
			},
			wantBody: `{"name":"the-bundle","description":"something","type":"bundle","source_url":"github.com/some-repo","access":"public","artifacts_schema":{"artifacts":"foo"},"connections_schema":{"connections":"bar"},"params_schema":{"params":{"hello":"world"}},"ui_schema":{"ui":"baz"},"spec":{"access":"public","artifacts":{"artifacts":"foo"},"connections":{"connections":"bar"},"description":"something","name":"the-bundle","params":{"params":{"hello":"world"}},"schema":"","source_url":"github.com/some-repo","steps":[{"path":"deploy","provisioner":"opentofu"}],"type":"bundle","ui":{"ui":"baz"}}}`,
		},
		{
			name:      "Submits an app block field if one does exist",
			guideType: "",
			bundle: bundle.Bundle{
				Name:        "the-bundle",
				Description: "something",
				SourceURL:   "github.com/some-repo",
				Type:        "bundle",
				Access:      "public",
				Steps: []bundle.Step{
					{
						Path:        "deploy",
						Provisioner: "opentofu",
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
			},
			wantBody: `{"name":"the-bundle","description":"something","type":"bundle","source_url":"github.com/some-repo","access":"public","artifacts_schema":{"artifacts":"foo"},"connections_schema":{"connections":"bar"},"params_schema":{"params":{"hello":"world"}},"ui_schema":{"ui":"baz"},"app":{"envs":{"LOG_LEVEL":"warn"},"policies":[".connections.vpc.data.infrastructure.arn"],"secrets":{"STRIPE_KEY":{"description":"Access key for live stripe accounts","json":false,"required":true,"title":"A secret"}}},"spec":{"access":"public","app":{"envs":{"LOG_LEVEL":"warn"},"policies":[".connections.vpc.data.infrastructure.arn"],"secrets":{"STRIPE_KEY":{"description":"Access key for live stripe accounts","json":false,"required":true,"title":"A secret"}}},"artifacts":{"artifacts":"foo"},"connections":{"connections":"bar"},"description":"something","name":"the-bundle","params":{"params":{"hello":"world"}},"schema":"","source_url":"github.com/some-repo","steps":[{"path":"deploy","provisioner":"opentofu"}],"type":"bundle","ui":{"ui":"baz"}}}`,
		},
		{
			name:      "Submits an operator.md guide if it exist",
			guideType: "md",
			bundle: bundle.Bundle{
				Name:        "the-bundle",
				Description: "something",
				SourceURL:   "github.com/some-repo",
				Type:        "bundle",
				Access:      "public",
				Steps: []bundle.Step{
					{
						Path:        "deploy",
						Provisioner: "opentofu",
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
			},
			wantBody: `{"name":"the-bundle","description":"something","type":"bundle","source_url":"github.com/some-repo","access":"public","artifacts_schema":{"artifacts":"foo"},"connections_schema":{"connections":"bar"},"params_schema":{"params":{"hello":"world"}},"ui_schema":{"ui":"baz"},"operator_guide":"IyBTb21lIE1hcmtkb3duIQo=","app":{"envs":{"LOG_LEVEL":"warn"},"policies":[".connections.vpc.data.infrastructure.arn"],"secrets":{"STRIPE_KEY":{"description":"Access key for live stripe accounts","json":false,"required":true,"title":"A secret"}}},"spec":{"access":"public","app":{"envs":{"LOG_LEVEL":"warn"},"policies":[".connections.vpc.data.infrastructure.arn"],"secrets":{"STRIPE_KEY":{"description":"Access key for live stripe accounts","json":false,"required":true,"title":"A secret"}}},"artifacts":{"artifacts":"foo"},"connections":{"connections":"bar"},"description":"something","name":"the-bundle","params":{"params":{"hello":"world"}},"schema":"","source_url":"github.com/some-repo","steps":[{"path":"deploy","provisioner":"opentofu"}],"type":"bundle","ui":{"ui":"baz"}}}`,
		},
		{
			name:      "Submits an operator.mdx guide if it exist",
			guideType: "mdx",
			bundle: bundle.Bundle{
				Name:        "the-bundle",
				Description: "something",
				SourceURL:   "github.com/some-repo",
				Type:        "bundle",
				Access:      "public",
				Steps: []bundle.Step{
					{
						Path:         "deploy",
						Provisioner:  "opentofu",
						SkipOnDelete: true,
						Config: map[string]interface{}{
							"foo": "bar",
						},
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
			},
			wantBody: `{"name":"the-bundle","description":"something","type":"bundle","source_url":"github.com/some-repo","access":"public","artifacts_schema":{"artifacts":"foo"},"connections_schema":{"connections":"bar"},"params_schema":{"params":{"hello":"world"}},"ui_schema":{"ui":"baz"},"operator_guide":"IyBTb21lIE1hcmtkb3duIQo=","app":{"envs":{"LOG_LEVEL":"warn"},"policies":[".connections.vpc.data.infrastructure.arn"],"secrets":{"STRIPE_KEY":{"description":"Access key for live stripe accounts","json":false,"required":true,"title":"A secret"}}},"spec":{"access":"public","app":{"envs":{"LOG_LEVEL":"warn"},"policies":[".connections.vpc.data.infrastructure.arn"],"secrets":{"STRIPE_KEY":{"description":"Access key for live stripe accounts","json":false,"required":true,"title":"A secret"}}},"artifacts":{"artifacts":"foo"},"connections":{"connections":"bar"},"description":"something","name":"the-bundle","params":{"params":{"hello":"world"}},"schema":"","source_url":"github.com/some-repo","steps":[{"config":{"foo":"bar"},"path":"deploy","provisioner":"opentofu","skip_on_delete":true}],"type":"bundle","ui":{"ui":"baz"}}}`,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			testDir := t.TempDir()
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

			publisher := &publish.Publisher{
				Bundle:     &tc.bundle,
				RestClient: c,
				BuildDir:   testDir,
			}

			err := mockfilesystem.SetupBundle(testDir)

			if err != nil {
				t.Fatal(err)
			}

			err = mockfilesystem.WithOperatorGuide(testDir, tc.guideType)

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
				Provisioner: "opentofu",
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

	buildDir := t.TempDir()

	err := mockfilesystem.SetupBundle(buildDir)

	if err != nil {
		t.Fatal(err)
	}

	err = mockfilesystem.WithFilesToIgnore(buildDir)

	if err != nil {
		t.Fatal(err)
	}

	publisher := &publish.Publisher{
		Bundle:   &b,
		BuildDir: buildDir,
	}

	var buf bytes.Buffer

	err = publisher.ArchiveBundle(&buf)

	if err != nil {
		t.Fatal(err)
	}

	extractTarGz(bytes.NewReader(buf.Bytes()), buildDir)

	wantSrc := []string{"main.tf"}
	wantTopLevel := []string{"deploy", "massdriver.yaml", "src"}

	assertDirContains(wantTopLevel, path.Join(buildDir, "/untar/bundle"), t)
	assertDirContains(wantSrc, path.Join(buildDir, "/untar/bundle/src"), t)
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

func assertDirContains(want []string, dir string, t *testing.T) {
	got := []string{}

	info, err := os.ReadDir(dir)

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

func extractTarGz(gzipStream io.Reader, extractPath string) {
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
			dirErr := os.MkdirAll(path.Join(extractPath, "/untar/", header.Name), 0755)
			if dirErr != nil {
				log.Fatalf("ExtractTarGz: Mkdir() failed: %s", err.Error())
			}
		case tar.TypeReg:
			outFile, headerErr := os.Create(path.Join(extractPath, "/untar/", header.Name))
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
