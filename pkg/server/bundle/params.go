package bundle

import (
	"path"

	"github.com/massdriver-cloud/mass/pkg/files"
)

const paramsFile = "_params.auto.tfvars.json"

// reconcileParams reads the params file keeping the md_metadata field intact as it's
// not represented in the UI yet, adds the incoming params, and writes the file back out.
func ReconcileParams(baseDir string, params map[string]any) error {
	paramPath := path.Join(baseDir, "src", paramsFile)

	fileParams := make(map[string]any)
	err := files.Read(paramPath, &fileParams)
	if err != nil {
		return err
	}

	combinedParams := make(map[string]any)
	if v, ok := fileParams["md_metadata"]; ok {
		combinedParams["md_metadata"] = v
	}

	for k, v := range params {
		combinedParams[k] = v
	}

	return files.Write(paramPath, combinedParams)
}
