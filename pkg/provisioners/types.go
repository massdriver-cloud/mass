package provisioners

import (
	"embed"
	"encoding/json"
)

//go:embed schemas/metadata-schema.json
var embedFS embed.FS

var MetadataSchema = parseMetadataSchema()

func parseMetadataSchema() map[string]interface{} {
	metadataBytes, err := embedFS.ReadFile("schemas/metadata-schema.json")
	if err != nil {
		return nil
	}

	var metadata map[string]interface{}
	err = json.Unmarshal(metadataBytes, &metadata)
	if err != nil {
		return nil
	}

	return metadata
}
