package provider

import (
	"github.com/hashicorp/go-cty/cty"
	"github.com/hashicorp/terraform-provider-kubernetes-alpha/tfplugin5"
	"github.com/pkg/errors"
)

// getConfigObjectType returns the type scaffolding for the provider config object.
func getConfigObjectType() cty.Type {
	return cty.Object(map[string]cty.Type{
		"host":                   cty.String,
		"username":               cty.String,
		"password":               cty.String,
		"client_certificate":     cty.String,
		"client_key":             cty.String,
		"cluster_ca_certificate": cty.String,
		"config_path":            cty.String,
		"config_context":         cty.String,
		"config_context_user":    cty.String,
		"config_context_cluster": cty.String,
		"token":                  cty.String,
		"insecure":               cty.Bool,
		"server_side_planning":   cty.Bool,
		"exec": cty.Object(map[string]cty.Type{
			"api_version": cty.String,
			"command":     cty.String,
			"env":         cty.Map(cty.String),
			"args":        cty.List(cty.String),
		}),
	})
}

// GetProviderConfigSchema contains the definitions of all configuration attributes
func GetProviderConfigSchema() (*tfplugin5.Schema, error) {
	b, err := ctyObjectToTfpluginSchema(getConfigObjectType())
	if err != nil {
		return nil, err
	}

	return &tfplugin5.Schema{
		Version: 1,
		Block:   b,
	}, nil
}

func ctyObjectToTfpluginSchema(o cty.Type) (*tfplugin5.Schema_Block, error) {
	b := tfplugin5.Schema_Block{}
	b.Attributes = []*tfplugin5.Schema_Attribute{}

	for k, v := range o.AttributeTypes() {
		tj, err := v.MarshalJSON()
		if err != nil {
			return nil, errors.Wrapf(err, "type %s fails marshalling to JSON", v.GoString())
		}
		b.Attributes = append(b.Attributes, &tfplugin5.Schema_Attribute{
			Name:     k,
			Type:     tj,
			Optional: true,
		})
	}

	return &b, nil
}
