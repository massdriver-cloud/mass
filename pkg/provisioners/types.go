package provisioners

import "strings"

// Provisioner defines the interface for infrastructure provisioner implementations.
type Provisioner interface {
	ExportMassdriverInputs(stepPath string, variables map[string]any) error
	ReadProvisionerInputs(stepPath string) (map[string]any, error)
	InitializeStep(stepPath string, sourcePath string) error
}

// NewProvisioner returns the appropriate Provisioner implementation for the given provisioner type string.
func NewProvisioner(provisionerType string) Provisioner {
	switch {
	case strings.Contains(provisionerType, "opentofu") || strings.Contains(provisionerType, "terraform"):
		return new(OpentofuProvisioner)
	case strings.Contains(provisionerType, "helm"):
		return new(HelmProvisioner)
	case strings.Contains(provisionerType, "bicep"):
		return new(BicepProvisioner)
	default:
		return new(NoopProvisioner)
	}
}

// NoopProvisioner is a no-op Provisioner used for unknown provisioner types.
type NoopProvisioner struct{}

// ExportMassdriverInputs is a no-op for unknown provisioner types.
func (p *NoopProvisioner) ExportMassdriverInputs(string, map[string]any) error {
	return nil
}

// ReadProvisionerInputs is a no-op for unknown provisioner types.
func (p *NoopProvisioner) ReadProvisionerInputs(string) (map[string]any, error) {
	return map[string]any{}, nil
}

// InitializeStep is a no-op for unknown provisioner types.
func (p *NoopProvisioner) InitializeStep(string, string) error {
	return nil
}
