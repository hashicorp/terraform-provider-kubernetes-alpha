package provider

import (
	"encoding/json"

	"github.com/davecgh/go-spew/spew"
	"github.com/pkg/errors"
	"github.com/zclconf/go-cty/cty"
	ctyjson "github.com/zclconf/go-cty/cty/json"
	"github.com/zclconf/go-cty/cty/msgpack"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/kubernetes/scheme"
)

// UnmarshalResource extracts a msgpack-ed resource into it's corresponding cty.Value
func UnmarshalResource(resource string, data []byte) (cty.Value, error) {
	// t, err := msgpack.ImpliedType(data)
	// if err != nil {
	// 	return cty.NullVal(cty.DynamicPseudoType), err
	// }
	s := GetProviderResourceSchema()
	t := GetObjectTypeFromSchema(s[resource])
	return msgpack.Unmarshal(data, t)
}

// MarshalResource extracts a msgpack-ed resource into it's corresponding cty.Value
func MarshalResource(resource string, data cty.Value) ([]byte, error) {
	// t, err := msgpack.ImpliedType(data)
	// if err != nil {
	// 	return cty.NullVal(cty.DynamicPseudoType), err
	// }
	s := GetProviderResourceSchema()
	t := GetObjectTypeFromSchema(s[resource])
	return msgpack.Marshal(data, t)
}

func ResourceFromYAMLManifest(manifest []byte) (map[string]interface{}, *schema.GroupVersionKind, error) {
	kdecoder := scheme.Codecs.UniversalDecoder()
	obj, gvk, err := kdecoder.Decode(manifest, nil, nil)
	if err != nil {
		return nil, nil, err
	}
	// convert the runtime.Object to unstructured.Unstructured
	unstruct, err := runtime.DefaultUnstructuredConverter.ToUnstructured(obj)
	if err != nil {
		return nil, nil, err
	}
	return unstruct, gvk, nil
}

func UnstructuredToCty(in map[string]interface{}) (*cty.Value, error) {
	jsonVal, err := json.Marshal(in)
	if err != nil {
		return nil, errors.Wrapf(err, "unable to marshal value")
	}

	simple := &ctyjson.SimpleJSONValue{}
	err = simple.UnmarshalJSON(jsonVal)
	if err != nil {
		return nil, errors.Wrapf(err, "unable to unmarshal to simple value")
	}
	return &simple.Value, nil
}

func CtyToUnstructured(in *cty.Value) (map[string]interface{}, error) {
	simple := &ctyjson.SimpleJSONValue{*in}
	jsonVal, err := simple.MarshalJSON()
	if err != nil {
		return nil, err
	}
	udata := map[string]interface{}{}
	err = json.Unmarshal(jsonVal, &udata)
	if err != nil {
		return nil, err
	}
	return udata, nil
}

// GVRFromCtyObject extracts a canonical schema.GroupVersionResource out of the resource's
// metadata by checking it agaings the discovery API via a RESTMapper
func GVRFromCtyObject(o *cty.Value) (*schema.GroupVersionResource, error) {
	m, err := GetRestMapper()
	if err != nil {
		return nil, err
	}
	apv := o.GetAttr("apiVersion").AsString()
	kind := o.GetAttr("kind").AsString()
	gv, err := schema.ParseGroupVersion(apv)
	if err != nil {
		return nil, err
	}
	gvr, err := m.ResourceFor(gv.WithResource(kind))
	Dlog.Printf("[GVRFromCtyObject] Discovered GVR: %s", spew.Sdump(gvr))
	if err != nil {
		return nil, err
	}
	return &gvr, nil
}
