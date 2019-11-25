package provider

import (
	"encoding/json"

	"github.com/pkg/errors"
	"github.com/zclconf/go-cty/cty"
	ctyjson "github.com/zclconf/go-cty/cty/json"
	"github.com/zclconf/go-cty/cty/msgpack"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/kubernetes/scheme"
)

// UnmarshalResource extracts a msgpack-ed resource into it's corresponding cty.Value
func UnmarshalResource(data []byte) (cty.Value, error) {
	t, err := msgpack.ImpliedType(data)
	if err != nil {
		return cty.NullVal(cty.DynamicPseudoType), err
	}

	return msgpack.Unmarshal(data, t)
}

// ExtractPackedManifest function expands the value of the manifest attribute from a MsgPack plan.
func ExtractPackedManifest(in []byte) (out string, err error) {
	r, err := UnmarshalResource(in)
	if err != nil {
		Dlog.Printf("[ExtractManifestFromPlan][UnmarshaledPlan] Failed to unmarshal msgpack: %s\n", err.Error())
		return
	}
	if r.IsNull() {
		return
	}
	m := r.GetAttr("manifest")
	if !m.IsNull() {
		out = m.AsString()
	}
	return
}

func ResourceFromManifest(manifest []byte) (map[string]interface{}, *schema.GroupVersionKind, error) {
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
