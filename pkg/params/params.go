package params

import (
	"encoding/json"

	"github.com/massdriver-cloud/airlock/pkg/helm"
	"github.com/massdriver-cloud/airlock/pkg/terraform"
	"sigs.k8s.io/yaml"
)

func GetFromPath(templateName, path string) (string, error) {
	if path == "" {
		return "", nil
	}

	var (
		paramSchema string
		err         error
	)

	switch templateName {
	case "terraform-module":
		paramSchema, err = terraform.TfToSchema(path)
		if err != nil {
			return "", err
		}
	case "helm-chart":
		paramSchema, err = helm.HelmToSchema(path)
		if err != nil {
			return "", err
		}
	default:
		return "", nil
	}

	var params map[string]any
	if err = json.Unmarshal([]byte(paramSchema), &params); err != nil {
		return "", err
	}

	props := map[string]any{
		"params": params,
	}
	out, err := yaml.Marshal(props)
	if err != nil {
		return "", err
	}

	return string(out), nil
}
