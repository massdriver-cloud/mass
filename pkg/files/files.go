package files

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/BurntSushi/toml"
	"github.com/hashicorp/hcl/v2/hclparse"
	ctyjson "github.com/zclconf/go-cty/cty/json"
	"sigs.k8s.io/yaml"
)

const UserRW = 0600

func Write(path string, data any) error {
	var formattedData []byte
	ext := filepath.Ext(path)

	switch ext {
	case ".json":
		json, err := json.MarshalIndent(data, "", "  ")
		formattedData = json
		if err != nil {
			return err
		}
	default:
		return fmt.Errorf("unsupported file type: %s", ext)
	}

	return os.WriteFile(path, formattedData, UserRW)
}

func Read(path string, v any) error {
	ext := filepath.Ext(path)

	contents, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	switch ext {
	case ".json":
		if err = json.Unmarshal(contents, &v); err != nil {
			return err
		}
	case ".toml":
		if _, err = toml.Decode(string(contents), v); err != nil {
			return err
		}
	case ".yaml":
		if err = yaml.Unmarshal(contents, &v); err != nil {
			return err
		}
	case ".tfvars":
		if err = decodeTFVars(path, &v); err != nil {
			return err
		}
	default:
		return fmt.Errorf("unsupported file type: %s", ext)
	}

	return nil
}

func decodeTFVars(path string, v any) error {
	contents, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	parser := hclparse.NewParser()
	file, diags := parser.ParseHCL(contents, path)
	if diags.HasErrors() {
		return fmt.Errorf("failed to parse HCL: %s", diags.Error())
	}

	attrs, diags := file.Body.JustAttributes()
	if diags.HasErrors() {
		return fmt.Errorf("failed to get attributes: %s", diags.Error())
	}

	result := make(map[string]interface{})
	for name, attr := range attrs {
		val, diags := attr.Expr.Value(nil)
		if diags.HasErrors() {
			return fmt.Errorf("failed to evaluate attribute %s: %s", name, diags.Error())
		}
		// Convert cty.Value to Go value using JSON marshaling
		jsonBytes, err := ctyjson.Marshal(val, val.Type())
		if err != nil {
			return fmt.Errorf("failed to marshal attribute %s: %w", name, err)
		}
		var goVal interface{}
		if err = json.Unmarshal(jsonBytes, &goVal); err != nil {
			return fmt.Errorf("failed to unmarshal attribute %s: %w", name, err)
		}
		result[name] = goVal
	}

	// Convert the map to JSON and unmarshal into the target
	jsonBytes, err := json.Marshal(result)
	if err != nil {
		return fmt.Errorf("failed to marshal to JSON: %w", err)
	}

	if err = json.Unmarshal(jsonBytes, &v); err != nil {
		return fmt.Errorf("failed to unmarshal JSON: %w", err)
	}

	return nil
}
