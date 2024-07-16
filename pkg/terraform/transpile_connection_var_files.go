package terraform

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/massdriver-cloud/mass/pkg/bundle"
)

func transpileConnectionVarFile(path string, b *bundle.Bundle) error {
	emptyConnections := checkEmptySchema(b.Connections)

	if emptyConnections {
		err := os.WriteFile(path, []byte("{}"), 0644)

		if err != nil {
			return err
		}

		return nil
	}

	existingConnectionsVars, err := getExistingVars(path)

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

	err = os.WriteFile(path, bytes, 0644)

	return err
}
