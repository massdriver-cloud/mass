package params

import (
	"path"

	"github.com/massdriver-cloud/airlock/pkg/bicep"
	"github.com/massdriver-cloud/airlock/pkg/helm"
	"github.com/massdriver-cloud/airlock/pkg/opentofu"
	"github.com/massdriver-cloud/airlock/pkg/schema"
	"sigs.k8s.io/yaml"
)

func GetFromPath(templateName, paramsPath string) (string, error) {
	if paramsPath == "" {
		return "", nil
	}

	var (
		paramSchema *schema.Schema
		err         error
	)

	switch templateName {
	case "terraform-module", "opentofu-module":
		paramSchema, err = opentofu.TofuToSchema(paramsPath)
		if err != nil {
			return "", err
		}
	case "helm-chart":
		paramSchema, err = helm.HelmToSchema(path.Join(paramsPath, "values.yaml"))
		if err != nil {
			return "", err
		}
	case "bicep-template":
		paramSchema, err = bicep.BicepToSchema(paramsPath)
		if err != nil {
			return "", err
		}
	default:
		return "", nil
	}

	props := map[string]any{
		"params": paramSchema,
	}
	out, err := yaml.Marshal(props)
	if err != nil {
		return "", err
	}

	return string(out), nil
}
