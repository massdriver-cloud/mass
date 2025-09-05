package bundle

import (
	"encoding/json"
	"fmt"
	"net/url"
	"slices"

	"github.com/massdriver-cloud/airlock/pkg/schema"
	"github.com/massdriver-cloud/mass/pkg/jsonschema"
	"github.com/massdriver-cloud/mass/pkg/prettylogs"
	"github.com/massdriver-cloud/mass/pkg/provisioners"
	"github.com/massdriver-cloud/massdriver-sdk-go/massdriver/client"
)

// LintSeverity represents the severity level of a lint issue
type LintSeverity int

const (
	// LintWarning represents a non-blocking issue that should be reported but not halt execution
	LintWarning LintSeverity = iota
	// LintError represents a blocking issue that should halt execution
	LintError
)

// String returns the string representation of LintSeverity
func (s LintSeverity) String() string {
	switch s {
	case LintWarning:
		return prettylogs.Orange("WARNING").String()
	case LintError:
		return prettylogs.Red("ERROR").String()
	default:
		return "UNKNOWN"
	}
}

// LintIssue represents a single lint issue with its severity and message
type LintIssue struct {
	Severity LintSeverity
	Message  string
	Rule     string // The name of the lint rule that generated this issue
}

// Error implements the error interface for LintIssue
func (i LintIssue) Error() string {
	return fmt.Sprintf("[%s]: %s", i.Severity, i.Message)
}

// LintResult holds the results of a linting operation
type LintResult struct {
	Issues []LintIssue
}

// AddError adds an error-level issue to the result
func (r *LintResult) AddError(rule, message string) {
	r.Issues = append(r.Issues, LintIssue{
		Severity: LintError,
		Message:  message,
		Rule:     rule,
	})
}

// AddWarning adds a warning-level issue to the result
func (r *LintResult) AddWarning(rule, message string) {
	r.Issues = append(r.Issues, LintIssue{
		Severity: LintWarning,
		Message:  message,
		Rule:     rule,
	})
}

// HasIssues returns true if the result contains any error-level issues
func (r *LintResult) HasIssues() bool {
	return len(r.Issues) > 0
}

// HasErrors returns true if the result contains any error-level issues
func (r *LintResult) HasErrors() bool {
	for _, issue := range r.Issues {
		if issue.Severity == LintError {
			return true
		}
	}
	return false
}

// HasWarnings returns true if the result contains any warning-level issues
func (r *LintResult) HasWarnings() bool {
	for _, issue := range r.Issues {
		if issue.Severity == LintWarning {
			return true
		}
	}
	return false
}

// Errors returns all error-level issues
func (r *LintResult) Errors() []LintIssue {
	var errors []LintIssue
	for _, issue := range r.Issues {
		if issue.Severity == LintError {
			errors = append(errors, issue)
		}
	}
	return errors
}

// Warnings returns all warning-level issues
func (r *LintResult) Warnings() []LintIssue {
	var warnings []LintIssue
	for _, issue := range r.Issues {
		if issue.Severity == LintWarning {
			warnings = append(warnings, issue)
		}
	}
	return warnings
}

// Merge combines this result with another result
func (r *LintResult) Merge(other LintResult) {
	r.Issues = append(r.Issues, other.Issues...)
}

// IsClean returns true if there are no issues at all
func (r *LintResult) IsClean() bool {
	return len(r.Issues) == 0
}

func (b *Bundle) LintSchema(mdClient *client.Client) LintResult {
	var result LintResult

	bundleSchemaURL, err := url.JoinPath(mdClient.Config.URL, "json-schemas", "bundle.json")
	if err != nil {
		result.AddError("schema-validation", fmt.Sprintf("failed to construct bundle schema URL: %v", err))
		return result
	}

	sch, err := jsonschema.LoadSchemaFromURL(bundleSchemaURL)
	if err != nil {
		result.AddError("schema-validation", fmt.Sprintf("failed to compile bundle schema: %v", err))
		return result
	}

	err = jsonschema.ValidateGo(sch, b)
	if err != nil {
		result.AddError("schema-validation", err.Error())
		return result
	}

	return result
}

func (b *Bundle) LintParamsConnectionsNameCollision() LintResult {
	var result LintResult

	if b.Params != nil {
		if params, ok := b.Params["properties"]; ok {
			if b.Connections != nil {
				if connections, connectionsOk := b.Connections["properties"]; connectionsOk {
					for param := range params.(map[string]any) {
						for connection := range connections.(map[string]any) {
							if param == connection {
								result.AddError("name-collision", fmt.Sprintf("a parameter and connection have the same name: %s", param))
							}
						}
					}
				}
			}
		}
	}
	return result
}

func (b *Bundle) LintMatchRequired() LintResult {
	var result LintResult

	jsonBytes, marshalErr := json.Marshal(b.Params)
	if marshalErr != nil {
		result.AddError("required-match", fmt.Sprintf("failed to marshal parameters: %v", marshalErr))
		return result
	}

	paramsSchema := schema.Schema{}
	unmarshalErr := paramsSchema.UnmarshalJSON(jsonBytes)
	if unmarshalErr != nil {
		result.AddError("required-match", fmt.Sprintf("failed to unmarshal parameters: %v", unmarshalErr))
		return result
	}

	err := matchRequired(&paramsSchema)
	if err != nil {
		result.AddError("required-match", err.Error())
	}

	return result
}

//nolint:gocognit
func matchRequired(sch *schema.Schema) error {
	expandedProperties := schema.ExpandProperties(sch)

	propertyNames := []string{}

	for pair := expandedProperties.Oldest(); pair != nil; pair = pair.Next() {
		propertyNames = append(propertyNames, pair.Key)
		prop := pair.Value
		if prop.Type == "object" || prop.Type == "" {
			err := matchRequired(prop)
			if err != nil {
				return err
			}
		}
	}

	for _, req := range sch.Required {
		if !slices.Contains(propertyNames, req) {
			return fmt.Errorf("required parameter %s is not defined in properties", req)
		}
	}

	return nil
}

//nolint:gocognit
func (b *Bundle) LintInputsMatchProvisioner() LintResult {
	var result LintResult

	massdriverInputs := b.CombineParamsConnsMetadata()
	massdriverInputsProperties, ok := massdriverInputs["properties"].(map[string]any)
	if !ok {
		result.AddError("param-mismatch", "enabled to convert to map[string]interface")
		return result
	}

	for _, step := range b.Steps {
		prov := provisioners.NewProvisioner(step.Provisioner)
		provisionerInputs, err := prov.ReadProvisionerInputs(step.Path)
		if err != nil {
			result.AddError("param-mismatch", err.Error())
			continue
		}
		// If this provisioner doesn't have "ReadProvisionerVariables" implemented, it returns nil
		if provisionerInputs == nil {
			continue
		}
		var provisionerInputsProperties map[string]any
		var exists bool
		if provisionerInputsProperties, exists = provisionerInputs["properties"].(map[string]any); !exists {
			provisionerInputsProperties = map[string]any{}
		}

		missingProvisionerInputs := []string{}
		for name := range massdriverInputsProperties {
			if _, exists = provisionerInputsProperties[name]; !exists {
				missingProvisionerInputs = append(missingProvisionerInputs, name)
			}
		}

		missingMassdriverInputs := []string{}
		for name := range provisionerInputsProperties {
			if _, exists = massdriverInputsProperties[name]; !exists {
				missingMassdriverInputs = append(missingMassdriverInputs, name)
			}
		}

		if len(missingMassdriverInputs) > 0 || len(missingProvisionerInputs) > 0 {
			errMsg := fmt.Sprintf("missing inputs detected in step %s:\n", step.Path)

			for _, p := range missingMassdriverInputs {
				errMsg += fmt.Sprintf("\t- input \"%s\" declared in IaC but missing massdriver.yaml declaration\n", p)
			}
			for _, v := range missingProvisionerInputs {
				errMsg += fmt.Sprintf("\t- input \"%s\" declared in massdriver.yaml but missing IaC declaration\n", v)
			}

			result.AddWarning("param-mismatch", errMsg)
		}
	}

	return result
}
