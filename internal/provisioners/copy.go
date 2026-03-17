package provisioners

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
)

// CopyFile copies a single file from source to destination.
func copyFile(src, dst string) error {
	sourceFileStat, err := os.Stat(src)
	if err != nil {
		return err
	}

	if !sourceFileStat.Mode().IsRegular() {
		return fmt.Errorf("%s is not a regular file", src)
	}

	source, err := os.Open(src)
	if err != nil {
		return err
	}
	defer source.Close()

	destination, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer destination.Close()

	_, err = io.Copy(destination, source)
	return err
}

// CopyDir copies a whole directory recursively from src to dst,
// ignoring files and directories that match any of the ignore patterns.
func copyDir(src string, dst string, ignorePatterns []string) error {
	entries, readErr := os.ReadDir(src)
	if readErr != nil {
		return readErr
	}

	for _, entry := range entries {
		srcPath := filepath.Join(src, entry.Name())
		dstPath := filepath.Join(dst, entry.Name())

		if shouldIgnore(srcPath, ignorePatterns) {
			continue
		}

		if entry.IsDir() {
			if err := os.MkdirAll(dstPath, entry.Type().Perm()); err != nil {
				return err
			}
			if err := copyDir(srcPath, dstPath, ignorePatterns); err != nil {
				return err
			}
		} else {
			if err := copyFile(srcPath, dstPath); err != nil {
				return err
			}
		}
	}
	return nil
}

// shouldIgnore checks if a given path matches any of the glob patterns.
func shouldIgnore(path string, patterns []string) bool {
	for _, pattern := range patterns {
		matched, err := filepath.Match(pattern, filepath.Base(path))
		if err != nil {
			fmt.Printf("Error matching pattern: %v\n", err)
			continue
		}
		if matched {
			return true
		}
	}
	return false
}
