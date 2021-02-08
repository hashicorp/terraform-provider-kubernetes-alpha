package provider

import (
	"github.com/hashicorp/terraform-plugin-go/tfprotov5"
	"github.com/hashicorp/terraform-plugin-go/tfprotov5/tftypes"
)

// GetProviderConfigSchema contains the definitions of all configuration attributes
func GetProviderConfigSchema() *tfprotov5.Schema {
	b := tfprotov5.SchemaBlock{
		Attributes: []*tfprotov5.SchemaAttribute{
			&tfprotov5.SchemaAttribute{
				Name:     "host",
				Optional: true,
				Type:     tftypes.String,
			},
			&tfprotov5.SchemaAttribute{
				Name:     "username",
				Optional: true,
				Type:     tftypes.String,
			},
			&tfprotov5.SchemaAttribute{
				Name:     "password",
				Optional: true,
				Type:     tftypes.String,
			},
			&tfprotov5.SchemaAttribute{
				Name:     "client_certificate",
				Optional: true,
				Type:     tftypes.String,
			},
			&tfprotov5.SchemaAttribute{
				Name:     "client_key",
				Optional: true,
				Type:     tftypes.String,
			},
			&tfprotov5.SchemaAttribute{
				Name:     "cluster_ca_certificate",
				Optional: true,
				Type:     tftypes.String,
			},
			&tfprotov5.SchemaAttribute{
				Name:     "config_path",
				Optional: true,
				Type:     tftypes.String,
			},
			&tfprotov5.SchemaAttribute{
				Name:     "config_context",
				Optional: true,
				Type:     tftypes.String,
			},
			&tfprotov5.SchemaAttribute{
				Name:     "config_context_user",
				Optional: true,
				Type:     tftypes.String,
			},
			&tfprotov5.SchemaAttribute{
				Name:     "config_context_cluster",
				Optional: true,
				Type:     tftypes.String,
			},
			&tfprotov5.SchemaAttribute{
				Name:     "token",
				Optional: true,
				Type:     tftypes.String,
			},
			&tfprotov5.SchemaAttribute{
				Name:     "insecure",
				Optional: true,
				Type:     tftypes.Bool,
			},
			&tfprotov5.SchemaAttribute{
				Name:     "exec",
				Optional: true,
				Type: tftypes.Object{
					AttributeTypes: map[string]tftypes.Type{
						"api_version": tftypes.String,
						"command":     tftypes.String,
						"env":         tftypes.Map{AttributeType: tftypes.String},
						"args":        tftypes.List{ElementType: tftypes.String},
					},
				},
			},
		},
	}

	return &tfprotov5.Schema{
		Version: 1,
		Block:   &b,
	}
}

// GetTypeFromSchema returns the equivalent tftypes.Type representation of a given tfprotov5.Schema
func GetTypeFromSchema(s *tfprotov5.Schema) tftypes.Type {
	schemaTypeAttributes := map[string]tftypes.Type{}
	for _, att := range s.Block.Attributes {
		schemaTypeAttributes[att.Name] = att.Type
	}
	return tftypes.Object{
		AttributeTypes: schemaTypeAttributes,
	}
}
