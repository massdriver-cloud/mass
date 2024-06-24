package bundle

import (
	"errors"

	"github.com/massdriver-cloud/mass/pkg/bicep"
	"github.com/massdriver-cloud/mass/pkg/terraform"
)

type InputHandler interface {
	ReadParams(string) ([]byte, error)
	WriteParams(string, map[string]any) error
	WriteSecrets(string, map[string]string) error
	ReadConnections(string) ([]byte, error)
	WriteConnections(string, map[string]any) error
}

func NewInputHandler(stepName string) (InputHandler, error) {
	switch stepName {
	case "terraform":
		return terraform.NewInputHandler(), nil
	case "bicep":
		return bicep.NewInputHandler(), nil
	}

	return nil, errors.New("unsupported provisioner")
}
