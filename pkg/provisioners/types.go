package provisioners

type Provisioner interface {
	ExportMassdriverInputs(stepPath string, variables map[string]interface{}) error
	ReadProvisionerInputs(stepPath string) (map[string]interface{}, error)
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

func (p *NoopProvisioner) ExportMassdriverInputs(stepPath string, variables map[string]interface{}) error {
	return nil
}
func (p *NoopProvisioner) ReadProvisionerInputs(stepPath string) (map[string]interface{}, error) {
	return nil, nil
}
