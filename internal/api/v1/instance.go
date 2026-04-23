package api

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/massdriver-cloud/massdriver-sdk-go/massdriver/client"
	"github.com/mitchellh/mapstructure"
)

// Instance represents a deployed bundle instance within a Massdriver environment.
type Instance struct {
	ID               string              `json:"id" mapstructure:"id"`
	Name             string              `json:"name" mapstructure:"name"`
	Status           string              `json:"status" mapstructure:"status"`
	Version          string              `json:"version" mapstructure:"version"`
	ReleaseStrategy  string              `json:"releaseStrategy" mapstructure:"releaseStrategy"`
	ResolvedVersion  string              `json:"resolvedVersion,omitempty" mapstructure:"resolvedVersion"`
	DeployedVersion  string              `json:"deployedVersion,omitempty" mapstructure:"deployedVersion"`
	AvailableUpgrade string              `json:"availableUpgrade,omitempty" mapstructure:"availableUpgrade"`
	Params           map[string]any      `json:"params,omitempty" mapstructure:"params"`
	Tags             map[string]string   `json:"tags,omitempty" mapstructure:"tags"`
	Cost             CostSummary         `json:"cost" mapstructure:"cost"`
	CreatedAt        time.Time           `json:"createdAt,omitempty" mapstructure:"createdAt"`
	UpdatedAt        time.Time           `json:"updatedAt,omitempty" mapstructure:"updatedAt"`
	StatePaths       []InstanceStatePath `json:"statePaths,omitempty" mapstructure:"statePaths"`
	Environment      *Environment        `json:"environment,omitempty" mapstructure:"environment,omitempty"`
	Bundle           *Bundle             `json:"bundle,omitempty" mapstructure:"bundle,omitempty"`
	Component        *Component          `json:"component,omitempty" mapstructure:"component,omitempty"`
}

// InstanceStatePath is a Terraform/OpenTofu state path for a deployment step.
type InstanceStatePath struct {
	StepName string `json:"stepName" mapstructure:"stepName"`
	StateURL string `json:"stateUrl" mapstructure:"stateUrl"`
}

// InstanceResource pairs a bundle output handle (field) with the resource that was produced on that handle.
type InstanceResource struct {
	Field    string   `json:"field" mapstructure:"field"`
	Resource Resource `json:"resource" mapstructure:"resource"`
}

// InstanceSecret holds metadata for a secret stored on an instance. The value is never returned.
type InstanceSecret struct {
	Name      string    `json:"name" mapstructure:"name"`
	CreatedAt time.Time `json:"createdAt" mapstructure:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt" mapstructure:"updatedAt"`
}

// ParamsJSON returns the instance parameters serialized as a pretty-printed JSON string.
func (inst *Instance) ParamsJSON() (string, error) {
	paramsJSON, err := json.MarshalIndent(inst.Params, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to marshal params to JSON: %w", err)
	}
	return string(paramsJSON), nil
}

// GetInstance retrieves an instance by ID from the Massdriver API.
func GetInstance(ctx context.Context, mdClient *client.Client, id string) (*Instance, error) {
	response, err := getInstance(ctx, mdClient.GQLv1, mdClient.Config.OrganizationID, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get instance %s: %w", id, err)
	}

	return toInstance(response.Instance)
}

// ListInstanceResources returns every output resource produced by the named instance, following pagination.
func ListInstanceResources(ctx context.Context, mdClient *client.Client, instanceID string) ([]InstanceResource, error) {
	var resources []InstanceResource
	var cursor *Cursor

	for {
		response, err := listInstanceResources(ctx, mdClient.GQLv1, mdClient.Config.OrganizationID, instanceID, cursor)
		if err != nil {
			return nil, fmt.Errorf("failed to list instance resources for %s: %w", instanceID, err)
		}

		for _, item := range response.Instance.Resources.Items {
			ir := InstanceResource{}
			if decodeErr := mapstructure.Decode(item, &ir); decodeErr != nil {
				return nil, fmt.Errorf("failed to decode instance resource: %w", decodeErr)
			}
			resources = append(resources, ir)
		}

		next := response.Instance.Resources.Cursor.Next
		if next == "" {
			break
		}
		cursor = &Cursor{Next: next}
	}

	return resources, nil
}

// ListInstances returns instances, optionally filtered.
func ListInstances(ctx context.Context, mdClient *client.Client, filter *InstancesFilter) ([]Instance, error) {
	response, err := listInstances(ctx, mdClient.GQLv1, mdClient.Config.OrganizationID, filter, nil, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to list instances: %w", err)
	}

	instances := make([]Instance, 0, len(response.Instances.Items))
	for _, resp := range response.Instances.Items {
		inst, instErr := toInstance(resp)
		if instErr != nil {
			return nil, fmt.Errorf("failed to convert instance: %w", instErr)
		}
		instances = append(instances, *inst)
	}

	return instances, nil
}

// UpdateInstance updates an instance's version constraint or release strategy.
func UpdateInstance(ctx context.Context, mdClient *client.Client, id string, input UpdateInstanceInput) (*Instance, error) {
	response, err := updateInstance(ctx, mdClient.GQLv1, mdClient.Config.OrganizationID, id, input)
	if err != nil {
		return nil, err
	}
	if !response.UpdateInstance.Successful {
		messages := response.UpdateInstance.GetMessages()
		if len(messages) > 0 {
			var sb strings.Builder
			sb.WriteString("unable to update instance:")
			for _, msg := range messages {
				sb.WriteString("\n  - ")
				sb.WriteString(msg.Message)
			}
			return nil, errors.New(sb.String())
		}
		return nil, errors.New("unable to update instance")
	}
	return toInstance(response.UpdateInstance.Result)
}

// SetInstanceSecret creates or updates a secret on an instance.
func SetInstanceSecret(ctx context.Context, mdClient *client.Client, id string, input SetInstanceSecretInput) (*InstanceSecret, error) {
	response, err := setInstanceSecret(ctx, mdClient.GQLv1, mdClient.Config.OrganizationID, id, input)
	if err != nil {
		return nil, err
	}
	if !response.SetInstanceSecret.Successful {
		messages := response.SetInstanceSecret.GetMessages()
		if len(messages) > 0 {
			var sb strings.Builder
			sb.WriteString("unable to set instance secret:")
			for _, msg := range messages {
				sb.WriteString("\n  - ")
				sb.WriteString(msg.Message)
			}
			return nil, errors.New(sb.String())
		}
		return nil, errors.New("unable to set instance secret")
	}
	return toInstanceSecret(response.SetInstanceSecret.Result)
}

// RemoveInstanceSecret removes a secret from an instance.
func RemoveInstanceSecret(ctx context.Context, mdClient *client.Client, id, name string) (*InstanceSecret, error) {
	response, err := removeInstanceSecret(ctx, mdClient.GQLv1, mdClient.Config.OrganizationID, id, name)
	if err != nil {
		return nil, err
	}
	if !response.RemoveInstanceSecret.Successful {
		messages := response.RemoveInstanceSecret.GetMessages()
		if len(messages) > 0 {
			var sb strings.Builder
			sb.WriteString("unable to remove instance secret:")
			for _, msg := range messages {
				sb.WriteString("\n  - ")
				sb.WriteString(msg.Message)
			}
			return nil, errors.New(sb.String())
		}
		return nil, errors.New("unable to remove instance secret")
	}
	return toInstanceSecret(response.RemoveInstanceSecret.Result)
}

func toInstance(v any) (*Instance, error) {
	inst := Instance{}
	if err := mapstructure.Decode(v, &inst); err != nil {
		return nil, fmt.Errorf("failed to decode instance: %w", err)
	}
	return &inst, nil
}

func toInstanceSecret(v any) (*InstanceSecret, error) {
	secret := InstanceSecret{}
	if err := mapstructure.Decode(v, &secret); err != nil {
		return nil, fmt.Errorf("failed to decode instance secret: %w", err)
	}
	return &secret, nil
}
