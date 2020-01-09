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

// ResourceUpdateObjectAttr is a cty.Transform callback that sets it's "object" attribute to a new cty.Value
func ResourceUpdateObjectAttr(newobj *cty.Value) func(path cty.Path, v cty.Value) (cty.Value, error) {
	return func(path cty.Path, v cty.Value) (cty.Value, error) {
		if path.Equals(cty.GetAttrPath("object")) {
			return *newobj, nil
		}
		return v, nil
	}
}

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
func MarshalResource(resource string, data *cty.Value) ([]byte, error) {
	// t, err := msgpack.ImpliedType(data)
	// if err != nil {
	// 	return cty.NullVal(cty.DynamicPseudoType), err
	// }
	s := GetProviderResourceSchema()
	t := GetObjectTypeFromSchema(s[resource])
	return msgpack.Marshal(*data, t)
}

// ResourceFromYAMLManifest parses a YAML Kubernetes manifest into unstructured client-go object plus a GroupVersionResource.
func ResourceFromYAMLManifest(manifest []byte) (map[string]interface{}, *schema.GroupVersionResource, error) {
	mapper, err := GetRestMapper()
	if err != nil {
		return nil, nil, err
	}
	kdec := scheme.Codecs.UniversalDeserializer()
	obj, gvk, err := kdec.Decode(manifest, nil, nil)
	if err != nil {
		return nil, nil, err
	}
	m, err := mapper.RESTMapping(gvk.GroupKind(), gvk.Version)
	if err != nil {
		return nil, nil, err
	}
	// convert the runtime.Object to unstructured.Unstructured
	unstruct, err := runtime.DefaultUnstructuredConverter.ToUnstructured(obj)
	if err != nil {
		return nil, nil, err
	}
	return unstruct, &m.Resource, nil
}

// UnstructuredToCty converts a Kubernetes dynamic client specific unstructured object
// into a Terraform specific cty.Object type manifest
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

// CtyToUnstructured converts a Terraform specific cty.Object type manifest
// into a Kubernetes dynamic client specific unstructured object
func CtyToUnstructured(in *cty.Value) (map[string]interface{}, error) {
	simple := &ctyjson.SimpleJSONValue{Value: *in}
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
func GVRFromCtyObject(o *cty.Value) (schema.GroupVersionResource, error) {
	r := schema.GroupVersionResource{}
	m, err := GetRestMapper()
	if err != nil {
		return r, err
	}
	apv := o.GetAttr("apiVersion").AsString()
	kind := o.GetAttr("kind").AsString()
	gv, err := schema.ParseGroupVersion(apv)
	if err != nil {
		return r, err
	}
	gvr, err := m.ResourceFor(gv.WithResource(kind))
	Dlog.Printf("[GVRFromCtyObject] Discovered GVR: %s", spew.Sdump(gvr))
	if err != nil {
		return r, err
	}
	return gvr, nil
}
