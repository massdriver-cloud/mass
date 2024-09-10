package provisioners

type Provisioner interface {
	ExportMassdriverInputs(stepPath string, variables map[string]interface{}) error
	ReadProvisionerInputs(stepPath string) (map[string]interface{}, error)
	InitializeStep(stepPath string, sourcePath string) error
}

func NewProvisioner(provisionerType string) Provisioner {
	switch provisionerType {
	case "opentofu", "terraform":
		return new(OpentofuProvisioner)
	case "bicep":
		return new(BicepProvisioner)
	default:
		return new(NoopProvisioner)
	}
}

type NoopProvisioner struct{}

func (p *NoopProvisioner) ExportMassdriverInputs(string, map[string]interface{}) error {
	return nil
}
func (p *NoopProvisioner) ReadProvisionerInputs(string) (map[string]interface{}, error) {
	return nil, nil
}
func (p *NoopProvisioner) InitializeStep(string, string) error {
	return nil
}
