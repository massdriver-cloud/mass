package bundle

import (
	"fmt"
	"maps"
	"os"
	"path"

	"github.com/massdriver-cloud/mass/pkg/provisioners"
	"gopkg.in/yaml.v3"
)

func ImportParams(buildPath string) error {
	b := Bundle{}
	var node yaml.Node

	fileBytes, err := os.ReadFile(path.Join(buildPath, "massdriver.yaml"))
	if err != nil {
		return err
	}

	err = yaml.Unmarshal(fileBytes, &node)
	if err != nil {
		return err
	}
	err = yaml.Unmarshal(fileBytes, &b)
	if err != nil {
		return err
	}

	var missing map[string]any
	for _, step := range stepsOrDefault(b.Steps) {
		prov := provisioners.NewProvisioner(step.Provisioner)
		inputs, err := prov.ReadProvisionerInputs(path.Join(buildPath, step.Path))
		if err != nil {
			return err
		}
		missing = provisioners.FindMissingFromMassdriver(inputs, b.CombineParamsConnsMetadata())
	}

	var encodedMissing yaml.Node
	encodedMissing.Encode(missing)

	var paramsNodePropertiesNodeValue *yaml.Node
	var paramsNodeRequiredNodeValue *yaml.Node
	for ii := 0; ii < len(node.Content[0].Content); ii += 2 {
		iiNodeName := node.Content[0].Content[ii]
		if iiNodeName.Value == "params" {
			paramsNodeValue := node.Content[0].Content[ii+1]
			for jj := 0; jj < len(paramsNodeValue.Content); jj += 2 {
				jjNodeName := paramsNodeValue.Content[jj]
				if jjNodeName.Value == "properties" {
					paramsNodePropertiesNodeValue = paramsNodeValue.Content[jj+1]
				}
				if jjNodeName.Value == "required" {
					paramsNodeRequiredNodeValue = paramsNodeValue.Content[jj+1]
				}
			}
			break
		}
	}

	var missingPropertiesNodeValue *yaml.Node
	var missingRequiredNodeValue *yaml.Node
	for kk := 0; kk < len(encodedMissing.Content); kk += 2 {
		kkNodeName := encodedMissing.Content[kk]
		if kkNodeName.Value == "properties" {
			missingPropertiesNodeValue = encodedMissing.Content[kk+1]
		}
		if kkNodeName.Value == "required" {
			missingRequiredNodeValue = encodedMissing.Content[kk+1]
		}
	}

	paramsNodePropertiesNodeValue.Content = append(paramsNodePropertiesNodeValue.Content, missingPropertiesNodeValue.Content...)
	paramsNodeRequiredNodeValue.Content = append(paramsNodeRequiredNodeValue.Content, missingRequiredNodeValue.Content...)

	newBytes, err := yaml.Marshal(&node)
	if err != nil {
		return err
	}

	fmt.Println(string(newBytes))

	return nil
}

// DON'T LET ME CHECK THIS IN, THIS IS A HACK
func stepsOrDefault(steps []Step) []Step {
	if steps == nil {
		return []Step{
			{Path: "src", Provisioner: "terraform"},
		}
	}

	return steps
}

func Foo() {
	foo := map[string]interface{}{
		"foo": map[string]interface{}{
			"bar": "baz",
		},
	}
	bar := map[string]interface{}{
		"foo": map[string]interface{}{
			"qux": "biz",
		},
	}
	poop := map[string]interface{}{}
	maps.Copy(poop, foo)
	maps.Copy(poop, bar)
	fmt.Println(poop)
}
