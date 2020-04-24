package provider

import (
	"github.com/hashicorp/go-cty/cty"
	"github.com/hashicorp/terraform-provider-kubernetes-alpha/tfplugin5"
)

// BlockMap is  a the basic building block of a configuration or resource object.
type BlockMap map[string]cty.Type

// GetConfigObjectType returns the type scaffolding for the provider config object.
func GetConfigObjectType() cty.Type {
	return cty.Object(BlockMap{
		"config_file":          cty.String,
		"server_side_planning": cty.Bool,
	})
}

// GetProviderConfigSchema contains the definitions of all configuration attributes
func GetProviderConfigSchema() *tfplugin5.Schema {
	cfgFileType, _ := cty.String.MarshalJSON()
	boolType, _ := cty.Bool.MarshalJSON()
	return &tfplugin5.Schema{
		Version: 1,
		Block: &tfplugin5.Schema_Block{
			Attributes: []*tfplugin5.Schema_Attribute{
				{
					Name:     "config_file",
					Type:     cfgFileType,
					Optional: true,
				},
				{
					Name:     "server_side_planning",
					Type:     boolType,
					Optional: true,
				},
			},
		},
	}
}
