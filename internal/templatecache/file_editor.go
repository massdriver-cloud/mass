package templatecache

import (
	"fmt"
	"io/fs"
	"os"
	"path"
	"strings"

	"text/template"

	"github.com/spf13/afero"
)

type FileEditor struct {
	fs                    afero.Fs
	readDirectory         string
	writeDirectory        string
	templateRootDirectory string
	templateData          *TemplateData
}

func (f *FileEditor) CopyTemplate() error {
	return afero.Walk(f.fs, f.readDirectory, func(filePath string, info fs.FileInfo, err error) error {
		if err != nil {
			return err
		}

		return f.mkDirOrWriteFile(filePath, info)
	})
}

func (f *FileEditor) mkDirOrWriteFile(filePath string, info fs.FileInfo) error {
	newPath := strings.Replace(filePath, f.readDirectory, "", 1)
	if newPath == "" {
		newPath = "."
	}

	if f.writeDirectory != "" && f.writeDirectory != "." {
		if _, err := f.fs.Stat(f.writeDirectory); err != nil {
			return f.fs.Mkdir(f.writeDirectory, 0755)
		}
	}

	outputPath := path.Join(f.writeDirectory, newPath)

	if info.IsDir() {
		if newPath == "." {
			return f.fs.MkdirAll(".", 0755)
		}

		return f.fs.Mkdir(outputPath, 0755)
	}

	file, err := afero.ReadFile(f.fs, filePath)

	if err != nil {
		return err
	}

	return f.promptAndWrite(file, outputPath)
}

func (f *FileEditor) promptAndWrite(file []byte, outputPath string) error {
	tmpl, errTmpl := template.New("tmpl").Delims("<md", "md>").Parse(string(file))
	if errTmpl != nil {
		return errTmpl
	}

	if _, err := os.Stat(outputPath); err == nil {
		fmt.Printf("%s exists. Overwrite? (y|N): ", outputPath)
		var response string
		fmt.Scanln(&response)

		if response == "y" || response == "Y" || response == "yes" {
			return f.writeToFile(outputPath, tmpl)
		}
	}

	return f.writeToFile(outputPath, tmpl)
}

func (f *FileEditor) writeToFile(outputPath string, tmpl *template.Template) error {
	outputFile, err := f.fs.Create(outputPath)

	if err != nil {
		return err
	}

	defer outputFile.Close()
	return tmpl.Execute(outputFile, f.templateData)
}
