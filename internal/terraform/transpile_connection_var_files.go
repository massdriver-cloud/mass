package terraform

import (
	"encoding/json"
	"fmt"

	"github.com/massdriver-cloud/mass/internal/bundle"
	"github.com/spf13/afero"
)

func transpileConnectionVarFile(path string, b *bundle.Bundle, fs afero.Fs) error {
	emptyConnections := checkEmptySchema(b.Connections)

	if emptyConnections {
		err := afero.WriteFile(fs, path, []byte("{}"), 0755)

		if err != nil {
			return err
		}

		return nil
	}

	existingConnectionsVars, err := getExistingVars(path, fs)

	if err != nil {
		return err
	}

	connectionsSchemaProperties, ok := b.Connections["properties"].(map[string]interface{})

	if !ok {
		return fmt.Errorf("expected connections schema properties to be an object")
	}

	values, _ := setValuesIfNotExists(connectionsSchemaProperties, existingConnectionsVars, nil)

	bytes, err := json.MarshalIndent(values, "", "    ")
	if err != nil {
		return err
	}

	err = afero.WriteFile(fs, path, bytes, 0755)

	return err
}
