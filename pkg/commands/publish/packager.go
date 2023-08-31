package publish

import (
	"archive/tar"
	"io"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/spf13/afero"
)

type CopyConfig struct {
	Allows  []string
	Ignores []string
}

type Packager struct {
	Filter *CopyConfig
	Fs     afero.Fs
}

func newPackager(filter *CopyConfig, fs afero.Fs) Packager {
	return Packager{
		Filter: filter,
		Fs:     fs,
	}
}

func (p *Packager) createArchiveWithFilter(dirPath string, prefix string, tarWriter *tar.Writer) error {
	absolutePath, pathErr := filepath.Abs(dirPath)

	if pathErr != nil {
		return pathErr
	}

	// walk through every file in the folder
	if err := afero.Walk(p.Fs, absolutePath, func(file string, fi os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		currentPath := strings.TrimPrefix(filepath.ToSlash(file), absolutePath)

		skip, errSkip := p.shouldSkip(fi, len(strings.Split(currentPath, "/")))

		if errSkip != nil {
			return errSkip
		}

		if skip {
			return nil
		}

		relativeDestinationFilePath := path.Join(prefix, currentPath)

		// generate tar header
		header, err := tar.FileInfoHeader(fi, file)

		if err != nil {
			return err
		}

		// must provide real name
		// (see https://golang.org/src/archive/tar/common.go?#L626)
		header.Name = relativeDestinationFilePath

		// write header
		if writeErr := tarWriter.WriteHeader(header); writeErr != nil {
			return writeErr
		}

		// if not a dir, write file content
		if !fi.IsDir() {
			data, openErr := p.Fs.Open(file)
			if openErr != nil {
				return openErr
			}
			if _, copyErr := io.Copy(tarWriter, data); copyErr != nil {
				return copyErr
			}
		}
		return nil
	}); err != nil {
		return err
	}

	return nil
}

func (p *Packager) shouldSkip(info fs.FileInfo, depth int) (bool, error) {
	name := info.Name()
	// if we're at the root of the bundle
	// we only want to honor the include list
	if depth == 2 && !p.shouldInclude(name) {
		if info.IsDir() {
			return true, filepath.SkipDir
		}
		return true, nil
	}

	// inside bundle directories like src, core-services, etc
	// we want to include every file _except_ the ones
	// that match the ignore _criteria_. File names, sizes, etc...
	if p.shouldIgnore(info) {
		if info.IsDir() {
			return true, filepath.SkipDir
		}
		return true, nil
	}
	return false, nil
}

func (p *Packager) shouldInclude(fileOrDirName string) bool {
	for _, allow := range p.Filter.Allows {
		if strings.Contains(fileOrDirName, allow) {
			return true
		}
	}
	return false
}

func (p *Packager) shouldIgnore(info fs.FileInfo) bool {
	fileName := info.Name()

	for _, ignore := range p.Filter.Ignores {
		if strings.Contains(fileName, ignore) {
			return true
		}
	}

	return false
}
