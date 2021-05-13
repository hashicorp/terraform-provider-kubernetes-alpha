package provider

import (
	"context"
	"fmt"
	"reflect"
	"testing"

	"github.com/hashicorp/go-hclog"
	"github.com/hashicorp/terraform-plugin-go/tfprotov5"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

func TestRemoveNulls(t *testing.T) {
	samples := []struct {
		in  map[string]interface{}
		out map[string]interface{}
	}{
		{
			in: map[string]interface{}{
				"foo": nil,
			},
			out: map[string]interface{}{},
		},
		{
			in: map[string]interface{}{
				"foo": nil,
				"bar": "test",
			},
			out: map[string]interface{}{
				"bar": "test",
			},
		},
		{
			in: map[string]interface{}{
				"foo": nil,
				"bar": []interface{}{nil, "test"},
			},
			out: map[string]interface{}{
				"bar": []interface{}{"test"},
			},
		},
		{
			in: map[string]interface{}{
				"foo": nil,
				"bar": []interface{}{
					map[string]interface{}{
						"some":  nil,
						"other": "data",
					},
					"test",
				},
			},
			out: map[string]interface{}{
				"bar": []interface{}{
					map[string]interface{}{
						"other": "data",
					},
					"test",
				},
			},
		},
	}

	for i, s := range samples {
		t.Run(fmt.Sprintf("sample%d", i+1), func(t *testing.T) {
			o := mapRemoveNulls(s.in)
			if !reflect.DeepEqual(s.out, o) {
				t.Fatal("sample and output are not equal")
			}
		})
	}
}

func TestLookUpGVKinCRDs(t *testing.T) {
	ps := RawProviderServer{logger: hclog.Default()}
	pCfgType := GetTypeFromSchema(GetProviderConfigSchema())
	cfgVal, err := tfprotov5.NewDynamicValue(
		pCfgType,
		tftypes.NewValue(pCfgType,
			map[string]tftypes.Value{
				"host":                   tftypes.NewValue(tftypes.String, nil),
				"username":               tftypes.NewValue(tftypes.String, nil),
				"password":               tftypes.NewValue(tftypes.String, nil),
				"client_certificate":     tftypes.NewValue(tftypes.String, nil),
				"client_key":             tftypes.NewValue(tftypes.String, nil),
				"cluster_ca_certificate": tftypes.NewValue(tftypes.String, nil),
				"config_path":            tftypes.NewValue(tftypes.String, "~/.kube/config"),
				"config_context":         tftypes.NewValue(tftypes.String, nil),
				"config_context_user":    tftypes.NewValue(tftypes.String, nil),
				"config_context_cluster": tftypes.NewValue(tftypes.String, nil),
				"token":                  tftypes.NewValue(tftypes.String, nil),
				"insecure":               tftypes.NewValue(tftypes.Bool, nil),
				"exec": tftypes.NewValue(tftypes.Object{
					AttributeTypes: map[string]tftypes.Type{
						"api_version": tftypes.String,
						"command":     tftypes.String,
						"env":         tftypes.Map{AttributeType: tftypes.String},
						"args":        tftypes.List{ElementType: tftypes.String},
					}}, nil),
			}),
	)
	if err != nil {
		t.Fatalf("%s", err)
	}
	ps.ConfigureProvider(
		context.Background(),
		&tfprotov5.ConfigureProviderRequest{
			Config: &cfgVal,
		},
	)
	schema, err := ps.lookUpGVKinCRDs(
		context.Background(),
		// schema.GroupVersionKind{"hashicorp.com", "v1", "TestCrd"},
		schema.GroupVersionKind{"hashicorp.com", "v1", "TestCrd"},
	)

	t.Logf("%+v", schema)
}
