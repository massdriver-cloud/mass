package params

import (
	"fmt"
	"path"

	"github.com/massdriver-cloud/airlock/pkg/bicep"
	"github.com/massdriver-cloud/airlock/pkg/helm"
	"github.com/massdriver-cloud/airlock/pkg/opentofu"
	"github.com/massdriver-cloud/airlock/pkg/result"
	"sigs.k8s.io/yaml"
)

func GetFromPath(templateName, paramsPath string) (string, error) {
	if paramsPath == "" {
		return "", nil
	}

	fmt.Printf("Importing params from %s...\n", paramsPath)
	var importResult result.SchemaResult
	switch templateName {
	case "terraform-module", "opentofu-module":
		importResult = opentofu.TofuToSchema(paramsPath)
	case "helm-chart":
		importResult = helm.HelmToSchema(path.Join(paramsPath, "values.yaml"))
	case "bicep-template":
		importResult = bicep.BicepToSchema(paramsPath)
	default:
		return "", nil
	}

	fmt.Print(importResult.PrettyDiags())

	if importResult.Schema == nil {
		fmt.Println("Params schema unable to be imported.")
		return "", fmt.Errorf("failed to import params schema")
	}

	props := map[string]any{
		"params": importResult.Schema,
	}
	content, err := yaml.Marshal(props)
	if err != nil {
		return "", err
	}
	fmt.Println("Params schema imported successfully.")

	return string(content), nil
}
