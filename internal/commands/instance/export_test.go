package instance_test

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/massdriver-cloud/mass/internal/api/v1"
	"github.com/massdriver-cloud/mass/internal/commands/instance"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
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

func (mbf *MockBundleFetcher) FetchBundle(ctx context.Context, bundleName, version, directory string) error {
	args := mbf.Called(ctx, bundleName, version, directory)
	mbf.FetchedBundles = append(mbf.FetchedBundles, bundleName)
	return args.Error(0)
}

type MockResourceLister struct {
	mock.Mock
}

func (mrl *MockResourceLister) ListInstanceResources(ctx context.Context, instanceID string) ([]api.InstanceResource, error) {
	args := mrl.Called(ctx, instanceID)
	resources, _ := args.Get(0).([]api.InstanceResource)
	return resources, args.Error(1)
}

type MockResourceExporter struct {
	mock.Mock
	ExportedResources []string
}

func (mre *MockResourceExporter) ExportResource(ctx context.Context, resourceID, format string) (string, error) {
	args := mre.Called(ctx, resourceID, format)
	mre.ExportedResources = append(mre.ExportedResources, resourceID)
	return args.String(0), args.Error(1)
}

type MockStateFetcher struct {
	mock.Mock
	FetchedStates []string
}

func (msf *MockStateFetcher) FetchState(ctx context.Context, stateURL string) (any, error) {
	args := msf.Called(ctx, stateURL)
	msf.FetchedStates = append(msf.FetchedStates, stateURL)
	return args.Get(0), args.Error(1)
}

func TestExportInstance(t *testing.T) {
	tests := []struct {
		name          string
		instance      *api.Instance
		baseDir       string
		setupMocks    func(*MockFileSystem, *MockBundleFetcher, *MockResourceLister, *MockResourceExporter, *MockStateFetcher)
		expectedDirs  []string
		expectedFiles []string
		wantErr       bool
	}{
		{
			name: "export provisioned instance with all components",
			instance: &api.Instance{
				ID:              "ecomm-prod-db",
				Status:          "PROVISIONED",
				DeployedVersion: "1.2.3",
				Params: map[string]any{
					"param1": "value1",
					"param2": 42,
				},
				Bundle:    &api.Bundle{Name: "test-bundle"},
				Component: &api.Component{ID: "db", Name: "Database"},
				StatePaths: []api.InstanceStatePath{
					{StepName: "src", StateURL: "https://api.example.com/state/ecomm-prod-db/src"},
				},
			},
			baseDir: "/tmp/export",
			setupMocks: func(mfs *MockFileSystem, mbf *MockBundleFetcher, mrl *MockResourceLister, mre *MockResourceExporter, msf *MockStateFetcher) {
				mfs.On("MkdirAll", "/tmp/export/db", os.FileMode(0755)).Return(nil)
				mfs.On("WriteFile", "/tmp/export/db/params.json", mock.Anything, os.FileMode(0644)).Return(nil)

				mbf.On("FetchBundle", mock.Anything, "test-bundle", "1.2.3", "/tmp/export/db").Return(nil)

				mrl.On("ListInstanceResources", mock.Anything, "ecomm-prod-db").Return([]api.InstanceResource{
					{Field: "output", Resource: api.Resource{ID: "resource-1", Name: "test-resource"}},
				}, nil)
				mre.On("ExportResource", mock.Anything, "resource-1", "json").Return(`{"data": "test"}`, nil)
				mfs.On("WriteFile", "/tmp/export/db/artifact_output.json", mock.Anything, os.FileMode(0644)).Return(nil)

				msf.On("FetchState", mock.Anything, "https://api.example.com/state/ecomm-prod-db/src").Return(map[string]any{"version": "1.0"}, nil)
				mfs.On("WriteFile", "/tmp/export/db/src.tfstate.json", mock.Anything, os.FileMode(0644)).Return(nil)
			},
			expectedDirs: []string{"/tmp/export/db"},
			expectedFiles: []string{
				"/tmp/export/db/params.json",
				"/tmp/export/db/artifact_output.json",
				"/tmp/export/db/src.tfstate.json",
			},
			wantErr: false,
		},
		{
			name: "skip state write when no state exists yet",
			instance: &api.Instance{
				ID:              "ecomm-prod-cache",
				Status:          "PROVISIONED",
				DeployedVersion: "0.1.0",
				Params: map[string]any{
					"param1": "value1",
				},
				Bundle:    &api.Bundle{Name: "cache-bundle"},
				Component: &api.Component{ID: "cache"},
				StatePaths: []api.InstanceStatePath{
					{StepName: "src", StateURL: "https://api.example.com/state/ecomm-prod-cache/src"},
				},
			},
			baseDir: "/tmp/export",
			setupMocks: func(mfs *MockFileSystem, mbf *MockBundleFetcher, mrl *MockResourceLister, mre *MockResourceExporter, msf *MockStateFetcher) {
				mfs.On("MkdirAll", "/tmp/export/cache", os.FileMode(0755)).Return(nil)
				mfs.On("WriteFile", "/tmp/export/cache/params.json", mock.Anything, os.FileMode(0644)).Return(nil)

				mbf.On("FetchBundle", mock.Anything, "cache-bundle", "0.1.0", "/tmp/export/cache").Return(nil)

				mrl.On("ListInstanceResources", mock.Anything, "ecomm-prod-cache").Return([]api.InstanceResource{}, nil)

				msf.On("FetchState", mock.Anything, "https://api.example.com/state/ecomm-prod-cache/src").Return(nil, instance.ErrNoState)
			},
			expectedDirs: []string{"/tmp/export/cache"},
			expectedFiles: []string{
				"/tmp/export/cache/params.json",
			},
			wantErr: false,
		},
		{
			name: "skip export for non-provisioned instance",
			instance: &api.Instance{
				ID:        "ecomm-prod-pending",
				Status:    "INITIALIZED",
				Bundle:    &api.Bundle{Name: "pending-bundle"},
				Component: &api.Component{ID: "pending"},
			},
			baseDir: "/tmp/export",
			setupMocks: func(mfs *MockFileSystem, mbf *MockBundleFetcher, mrl *MockResourceLister, mre *MockResourceExporter, msf *MockStateFetcher) {
			},
			expectedDirs:  []string{},
			expectedFiles: []string{},
			wantErr:       false,
		},
		{
			name:     "validation error for invalid instance",
			instance: &api.Instance{ID: "invalid-instance"},
			baseDir:  "/tmp/export",
			setupMocks: func(mfs *MockFileSystem, mbf *MockBundleFetcher, mrl *MockResourceLister, mre *MockResourceExporter, msf *MockStateFetcher) {
			},
			expectedDirs:  []string{},
			expectedFiles: []string{},
			wantErr:       true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockFS := &MockFileSystem{}
			mockBundleFetcher := &MockBundleFetcher{}
			mockResourceLister := &MockResourceLister{}
			mockResourceExporter := &MockResourceExporter{}
			mockStateFetcher := &MockStateFetcher{}

			tt.setupMocks(mockFS, mockBundleFetcher, mockResourceLister, mockResourceExporter, mockStateFetcher)

			config := instance.ExportInstanceConfig{
				FileSystem:       mockFS,
				BundleFetcher:    mockBundleFetcher,
				ResourceLister:   mockResourceLister,
				ResourceExporter: mockResourceExporter,
				StateFetcher:     mockStateFetcher,
			}

			ctx, cancel := context.WithTimeout(t.Context(), 30*time.Second)
			defer cancel()

			err := instance.ExportInstanceWithConfig(ctx, &config, tt.instance, tt.baseDir)

			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)

			mockFS.AssertExpectations(t)
			mockBundleFetcher.AssertExpectations(t)
			mockResourceLister.AssertExpectations(t)
			mockResourceExporter.AssertExpectations(t)
			mockStateFetcher.AssertExpectations(t)

			assert.ElementsMatch(t, tt.expectedDirs, mockFS.CreatedDirs)

			for _, expectedFile := range tt.expectedFiles {
				_, exists := mockFS.WrittenFiles[expectedFile]
				assert.True(t, exists, "Expected file %s to be written", expectedFile)
			}
		})
	}
}

func TestExportInstance_FileSystemError(t *testing.T) {
	inst := &api.Instance{
		ID:              "ecomm-prod-db",
		Status:          "PROVISIONED",
		DeployedVersion: "1.2.3",
		Bundle:          &api.Bundle{Name: "test-bundle"},
		Component:       &api.Component{ID: "db"},
	}

	mockFS := &MockFileSystem{}
	mockFS.On("MkdirAll", mock.Anything, mock.Anything).Return(os.ErrPermission)

	config := instance.ExportInstanceConfig{
		FileSystem:       mockFS,
		BundleFetcher:    &MockBundleFetcher{},
		ResourceLister:   &MockResourceLister{},
		ResourceExporter: &MockResourceExporter{},
		StateFetcher:     &MockStateFetcher{},
	}

	ctx := t.Context()
	err := instance.ExportInstanceWithConfig(ctx, &config, inst, "/tmp/export")

	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to create directory")
}
