package pkg_test

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/massdriver-cloud/mass/pkg/api"
	"github.com/massdriver-cloud/mass/pkg/commands/pkg"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// Mock implementations
type MockFileSystem struct {
	mock.Mock
	CreatedDirs  []string
	WrittenFiles map[string][]byte
}

func (mfs *MockFileSystem) MkdirAll(path string, perm os.FileMode) error {
	args := mfs.Called(path, perm)
	mfs.CreatedDirs = append(mfs.CreatedDirs, path)
	return args.Error(0)
}

func (mfs *MockFileSystem) WriteFile(filename string, data []byte, perm os.FileMode) error {
	args := mfs.Called(filename, data, perm)
	if mfs.WrittenFiles == nil {
		mfs.WrittenFiles = make(map[string][]byte)
	}
	mfs.WrittenFiles[filename] = data
	return args.Error(0)
}

type MockBundleFetcher struct {
	mock.Mock
	FetchedBundles []string
}

func (mbf *MockBundleFetcher) FetchBundle(ctx context.Context, bundleName string, directory string) error {
	args := mbf.Called(ctx, bundleName, directory)
	mbf.FetchedBundles = append(mbf.FetchedBundles, bundleName)
	return args.Error(0)
}

type MockArtifactDownloader struct {
	mock.Mock
	DownloadedArtifacts []string
}

func (mad *MockArtifactDownloader) DownloadArtifact(ctx context.Context, artifactID string) (string, error) {
	args := mad.Called(ctx, artifactID)
	mad.DownloadedArtifacts = append(mad.DownloadedArtifacts, artifactID)
	return args.String(0), args.Error(1)
}

type MockStateFetcher struct {
	mock.Mock
	FetchedStates []string
}

func (msf *MockStateFetcher) FetchState(ctx context.Context, packageID string, stepPath string) (any, error) {
	args := msf.Called(ctx, packageID, stepPath)
	msf.FetchedStates = append(msf.FetchedStates, packageID+"/"+stepPath)
	return args.Get(0), args.Error(1)
}

func TestExportPackage(t *testing.T) {
	tests := []struct {
		name          string
		pkg           *api.Package
		baseDir       string
		setupMocks    func(*MockFileSystem, *MockBundleFetcher, *MockArtifactDownloader, *MockStateFetcher)
		expectedDirs  []string
		expectedFiles []string
		wantErr       bool
	}{
		{
			name: "export provisioned package with all components",
			pkg: &api.Package{
				ID:         "pkg-123",
				NamePrefix: "test-package-0001",
				Status:     string(api.PackageStatusProvisioned),
				Params: map[string]any{
					"param1": "value1",
					"param2": 42,
				},
				Artifacts: []api.Artifact{
					{
						ID:    "artifact-1",
						Name:  "test-artifact",
						Field: "output",
					},
				},
				Bundle: &api.Bundle{
					Name: "test-bundle",
					Spec: map[string]any{
						"steps": []map[string]any{
							{
								"path":        "src",
								"provisioner": "terraform",
							},
						},
					},
					SpecVersion: "application/vnd.massdriver.bundle.v1+json",
				},
				Manifest: &api.Manifest{
					Slug: "test-manifest",
				},
			},
			baseDir: "/tmp/export",
			setupMocks: func(mfs *MockFileSystem, mbf *MockBundleFetcher, mad *MockArtifactDownloader, msf *MockStateFetcher) {
				// FileSystem expectations
				mfs.On("MkdirAll", "/tmp/export/test-manifest", os.FileMode(0755)).Return(nil)
				mfs.On("WriteFile", "/tmp/export/test-manifest/params.json", mock.Anything, os.FileMode(0644)).Return(nil)

				// Bundle fetcher expectations
				mbf.On("FetchBundle", mock.Anything, "test-bundle", "/tmp/export/test-manifest").Return(nil)

				// Artifact downloader expectations
				mad.On("DownloadArtifact", mock.Anything, "artifact-1").Return(`{"data": "test"}`, nil)
				mfs.On("WriteFile", "/tmp/export/test-manifest/artifact_output.json", mock.Anything, os.FileMode(0644)).Return(nil)

				// State fetcher expectations
				msf.On("FetchState", mock.Anything, "pkg-123", "src").Return(map[string]any{"version": "1.0"}, nil)
				mfs.On("WriteFile", "/tmp/export/test-manifest/src.tfstate.json", mock.Anything, os.FileMode(0644)).Return(nil)
			},
			expectedDirs: []string{"/tmp/export/test-manifest"},
			expectedFiles: []string{
				"/tmp/export/test-manifest/params.json",
				"/tmp/export/test-manifest/artifact_output.json",
			},
			wantErr: false,
		},
		{
			name: "skip bundle export if not OCI compliant and skip state if nil",
			pkg: &api.Package{
				ID:         "pkg-123",
				NamePrefix: "test-package-0001",
				Status:     string(api.PackageStatusProvisioned),
				Params: map[string]any{
					"param1": "value1",
					"param2": 42,
				},
				Artifacts: []api.Artifact{
					{
						ID:    "artifact-1",
						Name:  "test-artifact",
						Field: "output",
					},
				},
				Bundle: &api.Bundle{
					Name: "test-bundle",
					Spec: map[string]any{
						"steps": []map[string]any{
							{
								"path":        "src",
								"provisioner": "terraform",
							},
						},
					},
					SpecVersion: "application/vnd.massdriver.bundle.v0+json",
				},
				Manifest: &api.Manifest{
					Slug: "test-manifest",
				},
			},
			baseDir: "/tmp/export",
			setupMocks: func(mfs *MockFileSystem, mbf *MockBundleFetcher, mad *MockArtifactDownloader, msf *MockStateFetcher) {
				// FileSystem expectations
				mfs.On("MkdirAll", "/tmp/export/test-manifest", os.FileMode(0755)).Return(nil)
				mfs.On("WriteFile", "/tmp/export/test-manifest/params.json", mock.Anything, os.FileMode(0644)).Return(nil)

				// Bundle fetcher expectations
				mbf.AssertNumberOfCalls(t, "FetchBundle", 0) // Should not be called since bundle is not OCI compliant

				// Artifact downloader expectations
				mad.On("DownloadArtifact", mock.Anything, "artifact-1").Return(`{"data": "test"}`, nil)
				mfs.On("WriteFile", "/tmp/export/test-manifest/artifact_output.json", mock.Anything, os.FileMode(0644)).Return(nil)

				// State fetcher expectations
				msf.On("FetchState", mock.Anything, "pkg-123", "src").Return(nil, nil)
			},
			expectedDirs: []string{"/tmp/export/test-manifest"},
			expectedFiles: []string{
				"/tmp/export/test-manifest/params.json",
				"/tmp/export/test-manifest/artifact_output.json",
			},
			wantErr: false,
		},
		{
			name: "export external package with remote references only",
			pkg: &api.Package{
				ID:         "pkg-456",
				NamePrefix: "external-package-0001",
				Status:     string(api.PackageStatusExternal),
				RemoteReferences: []api.RemoteReference{
					{
						Artifact: api.Artifact{
							ID:    "remote-artifact-1",
							Field: "remote-output",
						},
					},
				},
				Bundle: &api.Bundle{
					Name: "external-bundle",
				},
				Manifest: &api.Manifest{
					Slug: "external-manifest",
				},
			},
			baseDir: "/tmp/export",
			setupMocks: func(mfs *MockFileSystem, mbf *MockBundleFetcher, mad *MockArtifactDownloader, msf *MockStateFetcher) {
				mfs.On("MkdirAll", "/tmp/export/external-manifest", os.FileMode(0755)).Return(nil)
				mad.On("DownloadArtifact", mock.Anything, "remote-artifact-1").Return(`{"remote": "data"}`, nil)
				mfs.On("WriteFile", "/tmp/export/external-manifest/artifact_remote-output.json", mock.Anything, os.FileMode(0644)).Return(nil)
			},
			expectedDirs: []string{"/tmp/export/external-manifest"},
			expectedFiles: []string{
				"/tmp/export/external-manifest/artifact_remote-output.json",
			},
			wantErr: false,
		},
		{
			name: "skip export for non-provisioned non-external package",
			pkg: &api.Package{
				ID:         "pkg-789",
				NamePrefix: "pending-package-0001",
				Status:     "PENDING",
				Bundle: &api.Bundle{
					Name: "pending-bundle",
				},
				Manifest: &api.Manifest{
					Slug: "pending-manifest",
				},
			},
			baseDir: "/tmp/export",
			setupMocks: func(mfs *MockFileSystem, mbf *MockBundleFetcher, mad *MockArtifactDownloader, msf *MockStateFetcher) {
				// No expectations - should skip export
			},
			expectedDirs:  []string{},
			expectedFiles: []string{},
			wantErr:       false,
		},
		{
			name: "validation error for invalid package",
			pkg: &api.Package{
				ID: "invalid-pkg",
				// Missing required fields
			},
			baseDir: "/tmp/export",
			setupMocks: func(mfs *MockFileSystem, mbf *MockBundleFetcher, mad *MockArtifactDownloader, msf *MockStateFetcher) {
				// No expectations - should fail validation
			},
			expectedDirs:  []string{},
			expectedFiles: []string{},
			wantErr:       true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup mocks
			mockFS := &MockFileSystem{}
			mockBundleFetcher := &MockBundleFetcher{}
			mockArtifactDownloader := &MockArtifactDownloader{}
			mockStateFetcher := &MockStateFetcher{}

			tt.setupMocks(mockFS, mockBundleFetcher, mockArtifactDownloader, mockStateFetcher)

			// Create config with mocks
			config := pkg.ExportPackageConfig{
				FileSystem:         mockFS,
				BundleFetcher:      mockBundleFetcher,
				ArtifactDownloader: mockArtifactDownloader,
				StateFetcher:       mockStateFetcher,
			}

			// Run the function
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()

			err := pkg.ExportPackageWithConfig(ctx, &config, tt.pkg, tt.baseDir)

			// Check error expectation
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)

			// Verify mock calls
			mockFS.AssertExpectations(t)
			mockBundleFetcher.AssertExpectations(t)
			mockArtifactDownloader.AssertExpectations(t)
			mockStateFetcher.AssertExpectations(t)

			// Verify directories were created
			assert.ElementsMatch(t, tt.expectedDirs, mockFS.CreatedDirs)

			// Verify files were written
			for _, expectedFile := range tt.expectedFiles {
				_, exists := mockFS.WrittenFiles[expectedFile]
				assert.True(t, exists, "Expected file %s to be written", expectedFile)
			}
		})
	}
}

func TestExportPackage_FileSystemError(t *testing.T) {
	pack := &api.Package{
		ID:         "pkg-123",
		NamePrefix: "test-package-0001",
		Status:     string(api.PackageStatusProvisioned),
		Bundle: &api.Bundle{
			Name: "test-bundle",
			Spec: map[string]any{},
		},
		Manifest: &api.Manifest{
			Slug: "test-manifest",
		},
	}

	mockFS := &MockFileSystem{}
	mockFS.On("MkdirAll", mock.Anything, mock.Anything).Return(os.ErrPermission)

	config := pkg.ExportPackageConfig{
		FileSystem:         mockFS,
		BundleFetcher:      &MockBundleFetcher{},
		ArtifactDownloader: &MockArtifactDownloader{},
		StateFetcher:       &MockStateFetcher{},
	}

	ctx := context.Background()
	err := pkg.ExportPackageWithConfig(ctx, &config, pack, "/tmp/export")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to create directory")
}
