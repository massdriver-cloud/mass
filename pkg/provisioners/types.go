package provisioners

type Provisioner interface {
	ExportMassdriverVariables(stepPath string, variables map[string]interface{}) error
	ReadProvisionerVariables(stepPath string) (map[string]interface{}, error)
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

func (p *NoopProvisioner) ExportMassdriverVariables(stepPath string, variables map[string]interface{}) error {
	return nil
}
func (p *NoopProvisioner) ReadProvisionerVariables(stepPath string) (map[string]interface{}, error) {
	return nil, nil
}
