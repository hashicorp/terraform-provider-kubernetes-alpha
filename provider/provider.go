package provider

import (
	"github.com/alexsomesan/terraform-provider-raw/tfplugin5"
	"github.com/zclconf/go-cty/cty"
)

var providerState map[string]interface{}

const (
	DynamicClient string = "DYNCLIENT"
)

// GetProviderState provides access to a global state storage structure.
func GetProviderState() map[string]interface{} {
	if providerState == nil {
		providerState = make(map[string]interface{})
	}
	return providerState
}

// BlockMap a the basic building block of a configuration or resource object.
type BlockMap map[string]cty.Type

// GetConfigObjectType returns the type scaffolding for the provider config object.
func GetConfigObjectType() cty.Type {
	return cty.Object(BlockMap{"config_file": cty.String})
}

// GetObjectTypeFromSchema returns the type scaffolding for the manifest resource object.
func GetObjectTypeFromSchema(schema *tfplugin5.Schema) cty.Type {
	bm := BlockMap{}
	for _, att := range schema.Block.Attributes {
		var t cty.Type
		err := t.UnmarshalJSON(att.Type)
		if err != nil {
			Dlog.Printf("Failed to unmarshall type %s\n", att.Type)
			return cty.NilType
		}
		bm[att.Name] = t
	}
	return cty.Object(bm)
}

// GetProviderResourceSchema contains the definitions of all supported resources
func GetProviderResourceSchema() map[string]*tfplugin5.Schema {
	oType, _ := cty.DynamicPseudoType.MarshalJSON()
	mType, _ := cty.String.MarshalJSON()
	return map[string]*tfplugin5.Schema{
		"raw_manifest": &tfplugin5.Schema{
			Version: 1,
			Block: &tfplugin5.Schema_Block{
				Attributes: []*tfplugin5.Schema_Attribute{
					&tfplugin5.Schema_Attribute{
						Name:     "manifest",
						Type:     mType,
						Required: true,
					},
					&tfplugin5.Schema_Attribute{
						Name:     "object",
						Type:     oType,
						Optional: true,
						Computed: true,
					},
				},
			},
		},
	}
}

// GetProviderConfigSchema contains the definitions of all configuration attributes
func GetProviderConfigSchema() *tfplugin5.Schema {
	cfgFileType, _ := cty.String.MarshalJSON()
	return &tfplugin5.Schema{
		Version: 1,
		Block: &tfplugin5.Schema_Block{
			Attributes: []*tfplugin5.Schema_Attribute{
				&tfplugin5.Schema_Attribute{
					Name:     "config_file",
					Type:     cfgFileType,
					Optional: true,
				},
			},
		},
	}
}
