package api

import (
	"context"
	"fmt"

	"github.com/massdriver-cloud/massdriver-sdk-go/massdriver/client"
	"github.com/mitchellh/mapstructure"
)

// Instance represents a deployed bundle instance within a Massdriver environment.
type Instance struct {
	ID              string       `json:"id" mapstructure:"id"`
	Name            string       `json:"name" mapstructure:"name"`
	Status          string       `json:"status" mapstructure:"status"`
	Version         string       `json:"version" mapstructure:"version"`
	ReleaseStrategy string       `json:"releaseStrategy" mapstructure:"releaseStrategy"`
	Environment     *Environment `json:"environment,omitempty" mapstructure:"environment,omitempty"`
	Bundle          *Bundle      `json:"bundle,omitempty" mapstructure:"bundle,omitempty"`
}

// GetInstance retrieves an instance by ID from the Massdriver API.
func GetInstance(ctx context.Context, mdClient *client.Client, id string) (*Instance, error) {
	response, err := getInstanceById(ctx, mdClient.GQL, mdClient.Config.OrganizationID, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get instance %s: %w", id, err)
	}

	return toInstance(response.Instance)
}

// ListInstances returns instances, optionally filtered.
func ListInstances(ctx context.Context, mdClient *client.Client, filter *InstancesFilter) ([]Instance, error) {
	response, err := getInstances(ctx, mdClient.GQL, mdClient.Config.OrganizationID, filter, nil, nil)
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

func toInstance(v any) (*Instance, error) {
	inst := Instance{}
	if err := mapstructure.Decode(v, &inst); err != nil {
		return nil, fmt.Errorf("failed to decode instance: %w", err)
	}
	return &inst, nil
}
