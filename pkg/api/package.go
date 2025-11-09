package api

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/massdriver-cloud/massdriver-sdk-go/massdriver/client"
	"github.com/mitchellh/mapstructure"
)

type Package struct {
	ID                string                    `json:"id"`
	Slug              string                    `json:"slug"`
	Status            string                    `json:"status"`
	Params            map[string]any            `json:"params"`
	ParamsSchema      map[string]any            `json:"paramsSchema,omitempty"`
	CreatedAt         time.Time                 `json:"createdAt,omitempty"`
	UpdatedAt         time.Time                 `json:"updatedAt,omitempty"`
	Version           string                    `json:"version,omitempty"`
	ResolvedVersion   string                    `json:"resolvedVersion,omitempty"`
	ReleaseStrategy   string                    `json:"releaseStrategy,omitempty"`
	DeployedVersion   string                    `json:"deployedVersion,omitempty"`
	AvailableUpgrade  string                    `json:"availableUpgrade,omitempty"`
	Artifacts         []Artifact                `json:"artifacts,omitempty"`
	RemoteReferences  []RemoteReference         `json:"remoteReferences,omitempty"`
	Connections       []Connection               `json:"connections,omitempty"`
	Bundle            *Bundle                   `json:"bundle,omitempty" mapstructure:"bundle,omitempty"`
	Manifest          *Manifest                 `json:"manifest" mapstructure:"manifest,omitempty"`
	Environment       *Environment               `json:"environment,omitempty" mapstructure:"environment,omitempty"`
	LatestDeployment  *Deployment                `json:"latestDeployment,omitempty"`
	ActiveDeployment  *Deployment                `json:"activeDeployment,omitempty"`
	Deployments       []Deployment               `json:"deployments,omitempty"`
	Alarms            []Alarm                    `json:"alarms,omitempty"`
	SecretFields      []SecretField              `json:"secretFields,omitempty"`
	Decommissionable  *PackageDeletionLifecycle  `json:"decommissionable,omitempty"`
	Cost              *Cost                      `json:"cost,omitempty"`
}

type Connection struct {
	ID          string    `json:"id,omitempty"`
	PackageField string   `json:"packageField,omitempty"`
	Artifact    *Artifact `json:"artifact,omitempty"`
	CreatedAt   time.Time `json:"createdAt,omitempty"`
	UpdatedAt   time.Time `json:"updatedAt,omitempty"`
}

type Alarm struct {
	ID                 string          `json:"id"`
	CloudResourceID    string          `json:"cloudResourceId"`
	DisplayName        string          `json:"displayName"`
	Namespace          string          `json:"namespace,omitempty"`
	Name               string          `json:"name,omitempty"`
	Statistic          string          `json:"statistic,omitempty"`
	Dimensions         []Dimension     `json:"dimensions,omitempty"`
	ComparisonOperator string          `json:"comparisonOperator,omitempty"`
	Threshold          float64         `json:"threshold,omitempty"`
	Period             int             `json:"period,omitempty"`
	CurrentState       *AlarmState     `json:"currentState,omitempty"`
}

type Dimension struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

type AlarmState struct {
	ID         string    `json:"id"`
	Status     string    `json:"status"`
	Message    string    `json:"message"`
	Notification map[string]any `json:"notification"`
	OccurredAt time.Time `json:"occurredAt"`
}

type SecretField struct {
	Name          string         `json:"name"`
	Required      bool           `json:"required"`
	JSON          bool           `json:"json"`
	Title         string         `json:"title,omitempty"`
	Description   string         `json:"description,omitempty"`
	ValueMetadata *SecretMetadata `json:"valueMetadata,omitempty"`
}

type SecretMetadata struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	SHA256    string    `json:"sha256"`
	CreatedAt time.Time `json:"createdAt"`
}

type PackageDeletionLifecycle struct {
	Result   bool     `json:"result"`
	Messages []string `json:"messages,omitempty"`
}

func (p *Package) ParamsJSON() (string, error) {
	paramsJSON, err := json.MarshalIndent(p.Params, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to marshal params to JSON: %w", err)
	}
	return string(paramsJSON), nil
}

func (p *Package) ParamsSchemaJSON() (string, error) {
	if p.ParamsSchema == nil {
		return "{}", nil
	}
	paramsSchemaJSON, err := json.MarshalIndent(p.ParamsSchema, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to marshal params schema to JSON: %w", err)
	}
	return string(paramsSchemaJSON), nil
}

func GetPackageByName(ctx context.Context, mdClient *client.Client, name string) (*Package, error) {
	response, err := getPackage(ctx, mdClient.GQL, mdClient.Config.OrganizationID, name)
	if err != nil {
		return nil, fmt.Errorf("error when querying for package %s - ensure your project, target and package abbreviations are correct:\n\t%w", name, err)
	}

	return toPackage(response.Package)
}

func toPackage(p any) (*Package, error) {
	// Type assert to the generated type
	genPkg, ok := p.(getPackagePackage)
	if !ok {
		// Fallback to mapstructure for flexibility
		pkg := Package{}
		if err := mapstructure.Decode(p, &pkg); err != nil {
			return nil, fmt.Errorf("failed to decode package: %w", err)
		}
		return &pkg, nil
	}

	// Convert bundle
	var bundle *Bundle
	if genPkg.Bundle.Id != "" {
		b, err := toBundle(genPkg.Bundle)
		if err == nil {
			bundle = b
		}
	}

	// Convert manifest
	var manifest *Manifest
	if genPkg.Manifest.Id != "" {
		m, err := toManifest(genPkg.Manifest)
		if err == nil {
			manifest = m
		}
	}

	// Convert environment
	var environment *Environment
	if genPkg.Environment.Id != "" {
		e, err := toEnvironment(genPkg.Environment)
		if err == nil {
			environment = e
			// Convert project if present
			if genPkg.Environment.Project.Id != "" {
				proj, err := toProject(genPkg.Environment.Project)
				if err == nil {
					environment.Project = proj
				}
			}
		}
	}

	// Convert artifacts
	artifacts := make([]Artifact, len(genPkg.Artifacts))
	for i, a := range genPkg.Artifacts {
		artifacts[i] = Artifact{
			ID:    a.Id,
			Name:  a.Name,
			Type:  a.Type,
			Field: a.Field,
		}
	}

	// Convert remote references
	remoteReferences := make([]RemoteReference, len(genPkg.RemoteReferences))
	for i, rr := range genPkg.RemoteReferences {
		remoteReferences[i] = RemoteReference{
			Artifact: Artifact{
				ID:    rr.Artifact.Id,
				Name:  rr.Artifact.Name,
				Type:  rr.Artifact.Type,
				Field: rr.Artifact.Field,
			},
		}
	}

	// Convert connections
	connections := make([]Connection, len(genPkg.Connections))
	for i, c := range genPkg.Connections {
		conn := Connection{
			ID:          c.Id,
			PackageField: c.PackageField,
			CreatedAt:   c.CreatedAt,
			UpdatedAt:   c.UpdatedAt,
		}
		if c.Artifact.Id != "" {
			conn.Artifact = &Artifact{
				ID:    c.Artifact.Id,
				Name:  c.Artifact.Name,
				Type:  c.Artifact.Type,
				Field: c.Artifact.Field,
			}
		}
		connections[i] = conn
	}

	// Convert deployments
	var latestDeployment *Deployment
	if genPkg.LatestDeployment.Id != "" {
		latestDeployment = toDeploymentFromLatest(genPkg.LatestDeployment)
	}

	var activeDeployment *Deployment
	if genPkg.ActiveDeployment.Id != "" {
		activeDeployment = toDeploymentFromActive(genPkg.ActiveDeployment)
	}

	deployments := make([]Deployment, len(genPkg.Deployments))
	for i, d := range genPkg.Deployments {
		deployments[i] = *toDeploymentFromList(d)
	}

	// Convert alarms
	alarms := make([]Alarm, len(genPkg.Alarms))
	for i, a := range genPkg.Alarms {
		alarm := Alarm{
			ID:                 a.Id,
			CloudResourceID:    a.CloudResourceId,
			DisplayName:        a.DisplayName,
			Namespace:           a.Namespace,
			Name:               a.Name,
			Statistic:          a.Statistic,
			ComparisonOperator: a.ComparisonOperator,
			Threshold:          a.Threshold,
			Period:             a.Period,
		}
		// Convert dimensions
		dimensions := make([]Dimension, len(a.Dimensions))
		for j, d := range a.Dimensions {
			dimensions[j] = Dimension{
				Name:  d.Name,
				Value: d.Value,
			}
		}
		alarm.Dimensions = dimensions
		// Convert current state
		if a.CurrentState.Id != "" {
			alarm.CurrentState = &AlarmState{
				ID:          a.CurrentState.Id,
				Status:      string(a.CurrentState.Status),
				Message:     a.CurrentState.Message,
				Notification: a.CurrentState.Notification,
				OccurredAt:  a.CurrentState.OccurredAt,
			}
		}
		alarms[i] = alarm
	}

	// Convert secret fields
	secretFields := make([]SecretField, len(genPkg.SecretFields))
	for i, sf := range genPkg.SecretFields {
		secretField := SecretField{
			Name:     sf.Name,
			Required: sf.Required,
			JSON:     sf.Json,
			Title:    sf.Title,
			Description: sf.Description,
		}
		if sf.ValueMetadata.Id != "" {
			secretField.ValueMetadata = &SecretMetadata{
				ID:        sf.ValueMetadata.Id,
				Name:      sf.ValueMetadata.Name,
				SHA256:    sf.ValueMetadata.Sha256,
				CreatedAt: sf.ValueMetadata.CreatedAt,
			}
		}
		secretFields[i] = secretField
	}

	// Convert decommissionable
	var decommissionable *PackageDeletionLifecycle
	if genPkg.Decommissionable.Result || len(genPkg.Decommissionable.Messages) > 0 {
		messages := make([]string, len(genPkg.Decommissionable.Messages))
		for i, m := range genPkg.Decommissionable.Messages {
			messages[i] = m.Message
		}
		decommissionable = &PackageDeletionLifecycle{
			Result:   genPkg.Decommissionable.Result,
			Messages: messages,
		}
	}

	// Convert cost
	var cost *Cost
	if genPkg.Cost.Monthly.Average.Amount > 0 || genPkg.Cost.Daily.Average.Amount > 0 {
		cost = &Cost{
			Monthly: &CostType{
				Average: &CostSummary{
					Amount: genPkg.Cost.Monthly.Average.Amount,
				},
			},
			Daily: &CostType{
				Average: &CostSummary{
					Amount: genPkg.Cost.Daily.Average.Amount,
				},
			},
		}
	}

	pkg := Package{
		ID:               genPkg.Id,
		Slug:             genPkg.Slug,
		Status:           string(genPkg.Status),
		Params:           genPkg.Params,
		ParamsSchema:     genPkg.ParamsSchema,
		CreatedAt:        genPkg.CreatedAt,
		UpdatedAt:        genPkg.UpdatedAt,
		Version:          genPkg.Version,
		ResolvedVersion:  genPkg.ResolvedVersion,
		ReleaseStrategy:  string(genPkg.ReleaseStrategy),
		DeployedVersion:  genPkg.DeployedVersion,
		AvailableUpgrade: genPkg.AvailableUpgrade,
		Artifacts:        artifacts,
		RemoteReferences: remoteReferences,
		Connections:      connections,
		Bundle:           bundle,
		Manifest:         manifest,
		Environment:     environment,
		LatestDeployment: latestDeployment,
		ActiveDeployment: activeDeployment,
		Deployments:      deployments,
		Alarms:           alarms,
		SecretFields:     secretFields,
		Decommissionable: decommissionable,
		Cost:             cost,
	}

	return &pkg, nil
}

func toDeploymentFromLatest(d getPackagePackageLatestDeployment) *Deployment {
	var lastTransitionedAt *time.Time
	if !d.LastTransitionedAt.IsZero() {
		lastTransitionedAt = &d.LastTransitionedAt
	}
	return &Deployment{
		ID:                d.Id,
		Status:            string(d.Status),
		Action:            string(d.Action),
		Version:           d.Version,
		Message:           d.Message,
		DeployedBy:        d.DeployedBy,
		CreatedAt:         d.CreatedAt,
		UpdatedAt:         d.UpdatedAt,
		LastTransitionedAt: lastTransitionedAt,
		ElapsedTime:       d.ElapsedTime,
	}
}

func toDeploymentFromActive(d getPackagePackageActiveDeployment) *Deployment {
	var lastTransitionedAt *time.Time
	if !d.LastTransitionedAt.IsZero() {
		lastTransitionedAt = &d.LastTransitionedAt
	}
	return &Deployment{
		ID:                d.Id,
		Status:            string(d.Status),
		Action:            string(d.Action),
		Version:           d.Version,
		Message:           d.Message,
		DeployedBy:        d.DeployedBy,
		CreatedAt:         d.CreatedAt,
		UpdatedAt:         d.UpdatedAt,
		LastTransitionedAt: lastTransitionedAt,
		ElapsedTime:       d.ElapsedTime,
	}
}

func toDeploymentFromList(d getPackagePackageDeploymentsDeployment) *Deployment {
	var lastTransitionedAt *time.Time
	if !d.LastTransitionedAt.IsZero() {
		lastTransitionedAt = &d.LastTransitionedAt
	}
	return &Deployment{
		ID:                d.Id,
		Status:            string(d.Status),
		Action:            string(d.Action),
		Version:           d.Version,
		Message:           d.Message,
		DeployedBy:        d.DeployedBy,
		CreatedAt:         d.CreatedAt,
		UpdatedAt:         d.UpdatedAt,
		LastTransitionedAt: lastTransitionedAt,
		ElapsedTime:       d.ElapsedTime,
	}
}

func ConfigurePackage(ctx context.Context, mdClient *client.Client, name string, params map[string]any) (*Package, error) {
	response, err := configurePackage(ctx, mdClient.GQL, mdClient.Config.OrganizationID, name, params)

	if err != nil {
		return nil, err
	}

	if response.ConfigurePackage.Successful {
		return toPackage(response.ConfigurePackage.Result)
	}

	return nil, NewMutationError("failed to configure package", response.ConfigurePackage.Messages)
}

func SetPackageVersion(ctx context.Context, mdClient *client.Client, id string, version string, releaseStrategy ReleaseStrategy) (*Package, error) {
	response, err := setPackageVersion(ctx, mdClient.GQL, mdClient.Config.OrganizationID, id, version, releaseStrategy)

	if err != nil {
		return nil, err
	}

	if response.SetPackageVersion.Successful {
		return toPackage(response.SetPackageVersion.Result)
	}

	return nil, NewMutationError("failed to set package version", response.SetPackageVersion.Messages)
}

func DecommissionPackage(ctx context.Context, mdClient *client.Client, id string, message string) (*Deployment, error) {
	response, err := decommissionPackage(ctx, mdClient.GQL, mdClient.Config.OrganizationID, id, message)

	if err != nil {
		return nil, err
	}

	if response.DecommissionPackage.Successful {
		return response.DecommissionPackage.Result.toDeployment(), nil
	}

	return nil, NewMutationError("failed to decommission package", response.DecommissionPackage.Messages)
}
