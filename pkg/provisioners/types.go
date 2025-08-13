package provisioners

import "strings"

type Provisioner interface {
	ExportMassdriverInputs(stepPath string, variables map[string]any) error
	ReadProvisionerInputs(stepPath string) (map[string]any, error)
	InitializeStep(stepPath string, sourcePath string) error
}

func NewProvisioner(provisionerType string) Provisioner {
	if strings.Contains(provisionerType, "opentofu") || strings.Contains(provisionerType, "terraform") {
		return new(OpentofuProvisioner)
	} else if strings.Contains(provisionerType, "helm") {
		return new(HelmProvisioner)
	} else if strings.Contains(provisionerType, "bicep") {
		return new(BicepProvisioner)
	}
	return new(NoopProvisioner)
}

type NoopProvisioner struct{}

func (p *NoopProvisioner) ExportMassdriverInputs(string, map[string]any) error {
	return nil
}
func (p *NoopProvisioner) ReadProvisionerInputs(string) (map[string]any, error) {
	return nil, nil
}
func (p *NoopProvisioner) InitializeStep(string, string) error {
	return nil
}
