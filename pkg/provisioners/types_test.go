package provisioners_test

import (
	"fmt"
	"testing"

	"github.com/massdriver-cloud/mass/pkg/provisioners"
)

func TestNewProvisioner(t *testing.T) {
	tests := []struct {
		provisionerType string
		expectedType    string
	}{
		{"opentofu", "*provisioners.OpentofuProvisioner"},
		{"terraform", "*provisioners.OpentofuProvisioner"},
		{"helm", "*provisioners.HelmProvisioner"},
		{"bicep", "*provisioners.BicepProvisioner"},
		{"blah", "*provisioners.NoopProvisioner"},
		{"opentofu:1.10", "*provisioners.OpentofuProvisioner"},
		{"012345678910.dkr.ecr.us-west-2.amazonaws.com/myorg/prov-opentofu", "*provisioners.OpentofuProvisioner"},
	}

	for _, tt := range tests {
		t.Run(tt.provisionerType, func(t *testing.T) {
			provisioner := provisioners.NewProvisioner(tt.provisionerType)
			if fmt.Sprintf("%T", provisioner) != tt.expectedType {
				t.Errorf("expected %s, got %T", tt.expectedType, provisioner)
			}
		})
	}
}
