package studio

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/massdriver-cloud/mass/pkg/templatecache"
)

const (
	// StudioUIURL is the URL to download the studio UI from GitHub releases
	StudioUIURL = "https://github.com/massdriver-cloud/massdriver-devtool-ui/releases/latest/download/devtool-ui.tar.gz"
	// StudioUIDir is the subdirectory name for the UI files
	StudioUIDir = "studio-ui"
)

// SetupUIDir creates the base directory for the studio UI
func SetupUIDir() (string, error) {
	massDir, err := templatecache.GetOrCreateMassDir()
	if err != nil {
		return "", err
	}

	studioUIDir := path.Join(massDir, StudioUIDir)

	// TODO: Add version checking so we don't always wipe the dir
	if _, err = os.Stat(studioUIDir); err == nil {
		slog.Debug("Cleaning up UI dir")
		if err = os.RemoveAll(studioUIDir); err != nil {
			slog.Warn("Error cleaning up UI dir", "error", err)
		}
	}

	return studioUIDir, os.MkdirAll(studioUIDir, os.ModePerm)
}

// DownloadUI downloads and extracts the studio UI files
func DownloadUI(ctx context.Context, baseDir string) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, StudioUIURL, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to download UI: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to download UI: status %d", resp.StatusCode)
	}

	r, err := gzip.NewReader(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to create gzip reader: %w", err)
	}
	defer r.Close()

	tarReader := tar.NewReader(r)

	for {
		header, err := tarReader.Next()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return fmt.Errorf("failed to read tar: %w", err)
		}

		targetPath, err := sanitizeArchivePath(baseDir, header.Name)
		if err != nil {
			return err
		}

		info := header.FileInfo()
		if info.IsDir() {
			if err = os.MkdirAll(targetPath, info.Mode()); err != nil {
				return fmt.Errorf("failed to create directory: %w", err)
			}
			continue
		}

		if err = os.MkdirAll(filepath.Dir(targetPath), os.ModePerm); err != nil {
			return fmt.Errorf("failed to create parent directory: %w", err)
		}

		file, err := os.Create(targetPath)
		if err != nil {
			return fmt.Errorf("failed to create file: %w", err)
		}

		// Ignore the gosec linting, we are pulling from our repo only
		_, err = io.Copy(file, tarReader) // #nosec G110
		file.Close()
		if err != nil {
			return fmt.Errorf("failed to copy file contents: %w", err)
		}
	}

	return nil
}

// sanitizeArchivePath prevents path traversal attacks (zip-slip)
func sanitizeArchivePath(baseDir, targetPath string) (string, error) {
	fullPath := filepath.Join(baseDir, targetPath)
	if !strings.HasPrefix(fullPath, filepath.Clean(baseDir)) {
		return "", fmt.Errorf("invalid archive path: %s", targetPath)
	}
	return fullPath, nil
}
