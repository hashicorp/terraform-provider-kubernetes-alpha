package provider

import (
	"fmt"

	"github.com/alexsomesan/terraform-provider-kubedynamic/tfplugin5"
	"github.com/zclconf/go-cty/cty"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/rest"
)

var providerState map[string]interface{}

// keys into the provider state storage
const (
	DynamicClient   string = "DYNAMICCLIENT"
	DiscoveryClient string = "DISCOVERYCLIENT"
	RestClient      string = "RESTCLIENT"
	RestMapper      string = "RESTMAPPER"
)

// GetProviderState returns a global state storage structure.
func GetProviderState() map[string]interface{} {
	if providerState == nil {
		providerState = make(map[string]interface{})
	}
	return providerState
}

// GetDynamicClient returns a configured unstructured (dynamic) client instance
func GetDynamicClient() (dynamic.Interface, error) {
	s := GetProviderState()
	c, ok := s[DynamicClient]
	if !ok {
		return nil, fmt.Errorf("no dynamic client configured")
	}
	return c.(dynamic.Interface), nil
}

// GetDiscoveryClient returns a configured discyovery client instance
func GetDiscoveryClient() (discovery.DiscoveryInterface, error) {
	s := GetProviderState()
	c, ok := s[DiscoveryClient]
	if !ok {
		return nil, fmt.Errorf("no discovery client configured")
	}
	return c.(discovery.DiscoveryInterface), nil
}

// GetRestMapper returns a RESTMapper client instance
func GetRestMapper() (meta.RESTMapper, error) {
	s := GetProviderState()
	c, ok := s[RestMapper]
	if !ok {
		return nil, fmt.Errorf("no REST mapper client configured")
	}
	return c.(meta.RESTMapper), nil
}

// GetRestClient returns a raw REST client instance
func GetRestClient() (rest.Interface, error) {
	s := GetProviderState()
	c, ok := s[RestClient]
	if !ok {
		return nil, fmt.Errorf("no REST client client configured")
	}
	return c.(rest.Interface), nil
}

// BlockMap is  a the basic building block of a configuration or resource object.
type BlockMap map[string]cty.Type

// GetConfigObjectType returns the type scaffolding for the provider config object.
func GetConfigObjectType() cty.Type {
	return cty.Object(BlockMap{"config_file": cty.String})
}

// GetObjectTypeFromSchema returns a cty.Type that can wholy represent the schema input
func GetObjectTypeFromSchema(schema *tfplugin5.Schema) (cty.Type, error) {
	bm := BlockMap{}
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
func GetProviderResourceSchema() map[string]*tfplugin5.Schema {
	oType, _ := cty.DynamicPseudoType.MarshalJSON()
	mType, _ := cty.String.MarshalJSON()
	return map[string]*tfplugin5.Schema{
		"kubedynamic_yaml_manifest": &tfplugin5.Schema{
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
		"kubedynamic_hcl_manifest": &tfplugin5.Schema{
			Version: 1,
			Block: &tfplugin5.Schema_Block{
				Attributes: []*tfplugin5.Schema_Attribute{
					&tfplugin5.Schema_Attribute{
						Name:     "manifest",
						Type:     oType,
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
