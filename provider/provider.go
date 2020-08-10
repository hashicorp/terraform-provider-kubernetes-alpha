package provider

import (
	"fmt"

	"github.com/hashicorp/go-cty/cty"
	"github.com/hashicorp/terraform-provider-kubernetes-alpha/tfplugin5"
)

// GetObjectTypeFromSchema returns a cty.Type that can wholy represent the schema input
func GetObjectTypeFromSchema(schema *tfplugin5.Schema) (cty.Type, error) {
	bm := map[string]cty.Type{
		"wait_for": cty.Object(map[string]cty.Type{
			"fields": cty.Map(cty.String),
		}),
	}

	for _, att := range schema.Block.Attributes {
		var t cty.Type
		err := t.UnmarshalJSON(att.Type)
		if err != nil {
			return cty.NilType, fmt.Errorf("failed to unmarshall type %s", string(att.Type))
		}
		bm[att.Name] = t
	}

	return cty.Object(bm), nil
}

// GetProviderResourceSchema contains the definitions of all supported resources
func GetProviderResourceSchema() (map[string]*tfplugin5.Schema, error) {
	oType, err := cty.DynamicPseudoType.MarshalJSON()
	if err != nil {
		return nil, err
	}

	mapType, err := cty.Map(cty.String).MarshalJSON()
	if err != nil {
		return nil, err
	}

	return map[string]*tfplugin5.Schema{
		"kubernetes_manifest": {
			Version: 1,
			Block: &tfplugin5.Schema_Block{
				BlockTypes: []*tfplugin5.Schema_NestedBlock{
					{
						TypeName: "wait_for",
						Nesting:  tfplugin5.Schema_NestedBlock_SINGLE,
						MaxItems: 1,
						MinItems: 0,
						Block: &tfplugin5.Schema_Block{
							Attributes: []*tfplugin5.Schema_Attribute{
								{
									Name:     "fields",
									Type:     mapType,
									Optional: true,
								},
							},
						},
					},
				},
				Attributes: []*tfplugin5.Schema_Attribute{
					{
						Name:     "manifest",
						Type:     oType,
						Required: true,
					},
					{
						Name:     "object",
						Type:     oType,
						Optional: true,
						Computed: true,
					},
				},
			},
		},
	}, nil
}
