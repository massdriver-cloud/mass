package bundle

import (
	"maps"
)

func (b *Bundle) CombineParamsConnsMetadata() *Schema {
	combined := new(Schema)

	combined.Properties = make(map[string]*Schema)
	combined.Required = []string{}

	for _, sch := range []*Schema{b.Params, b.Connections, MetadataSchema} {
		maps.Copy(combined.Properties, sch.Properties)
		combined.Required = append(combined.Required, sch.Required...)
	}

	return combined
}
