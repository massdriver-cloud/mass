package beta

const PROVISIONER_ACTION_APPLY = 1
const PROVISIONER_ACTION_DESTROY = 2

type ProvisioningMetadata struct {
	Name string            `json:"name"`
	Tags map[string]string `json:"tags"`
}
