package terraform

import (
	"encoding/json"

	"github.com/massdriver-cloud/mass/pkg/bundle"
	"github.com/massdriver-cloud/mass/pkg/inputvars"
	"github.com/spf13/afero"
)

func transpileConnectionVarFile(path string, b *bundle.Bundle, fs afero.Fs) error {
	values, err := inputvars.TranspileConnectionVarFile(path, b, fs)

	if err != nil {
		return err
	}

	bytes, err := json.MarshalIndent(values, "", "    ")

	if err != nil {
		return err
	}

	err = afero.WriteFile(fs, path, bytes, 0755)

	return err
}
