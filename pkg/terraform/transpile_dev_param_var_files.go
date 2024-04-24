package terraform

import (
	"encoding/json"

	"github.com/massdriver-cloud/mass/pkg/bundle"
	"github.com/massdriver-cloud/mass/pkg/inputvars"
	"github.com/spf13/afero"
)

func transpileAndWriteDevParams(path string, b *bundle.Bundle, fs afero.Fs) error {
	result, err := inputvars.TranspileDevParams(path, b, fs)

	if err != nil {
		return err
	}

	resultWithMdMetadata := mergeMdMetadata(result, b.Name)

	bytes, err := json.MarshalIndent(resultWithMdMetadata, "", "    ")

	if err != nil {
		return err
	}

	err = afero.WriteFile(fs, path, bytes, 0755)

	return err
}

func mergeMdMetadata(params map[string]interface{}, bundleName string) map[string]interface{} {
	defaultMetadata := inputvars.DefaultMdMetadata(bundleName)

	// if md_metadata is not set, initialize it to a reasonable starting point
	if _, ok := params["md_metadata"]; !ok {
		params["md_metadata"] = defaultMetadata
	} else {
		// merge md metadata ties go to existing values
		for k, v := range defaultMetadata {
			if _, ok2 := params["md_metadata"].(map[string]interface{})[k]; !ok2 {
				params["md_metadata"].(map[string]interface{})[k] = v
			}
		}
	}

	return params
}
