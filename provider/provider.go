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

// GetPlanObjectType returns the type scaffolding for the manifest resource object.
func GetPlanObjectType() cty.Type {
	return cty.Object(BlockMap{"manifest": cty.DynamicPseudoType})
}

// GetProviderResourceSchema contains the definitions of all supported resources
func GetProviderResourceSchema() map[string]*tfplugin5.Schema {
	mType, _ := cty.DynamicPseudoType.MarshalJSON()
	sType, _ := cty.String.MarshalJSON()
	return map[string]*tfplugin5.Schema{
		"raw_manifest": &tfplugin5.Schema{
			Version: 1,
			Block: &tfplugin5.Schema_Block{
				Attributes: []*tfplugin5.Schema_Attribute{
					&tfplugin5.Schema_Attribute{
						Name:     "manifest",
						Type:     sType,
						Required: true,
					},
					&tfplugin5.Schema_Attribute{
						Name:     "object",
						Type:     mType,
						Optional: true,
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
