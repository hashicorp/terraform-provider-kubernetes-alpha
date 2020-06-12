package provider

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/hashicorp/go-cty/cty"
	ctyjson "github.com/hashicorp/go-cty/cty/json"
	"github.com/hashicorp/go-cty/cty/msgpack"
	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

// ResourceBulkUpdateObjectAttr is a cty.Transform callback that sets its "object" attribute to a new cty.Value
func ResourceBulkUpdateObjectAttr(newobj *cty.Value) func(path cty.Path, v cty.Value) (cty.Value, error) {
	return func(path cty.Path, v cty.Value) (cty.Value, error) {
		if path.Equals(cty.GetAttrPath("object")) {
			return *newobj, nil
		}
		return v, nil
	}
}

// ResourceDeepUpdateObjectAttr is a cty.Transform callback that sets each leaf node below the "object" attribute to a new cty.Value
func ResourceDeepUpdateObjectAttr(prefix cty.Path, newobj *cty.Value) func(path cty.Path, v cty.Value) (cty.Value, error) {
	return func(path cty.Path, v cty.Value) (cty.Value, error) {
		if !path.HasPrefix(prefix) {
			return v, nil
		}
		var objpath cty.Path = path[len(prefix):]

		newValForPath, err := objpath.Apply(*newobj)
		if err != nil {
			return v, nil
		}

		return newValForPath, nil
	}
}

// UnmarshalResource extracts a msgpack-ed resource into its corresponding cty.Value
func UnmarshalResource(resource string, data []byte) (cty.Value, error) {
	s, err := GetProviderResourceSchema()
	if err != nil {
		return cty.NilVal, err
	}
	t, err := GetObjectTypeFromSchema(s[resource])
	if err != nil {
		return cty.NilVal, err
	}
	return msgpack.Unmarshal(data, t)
}

// MarshalResource extracts a msgpack-ed resource into its corresponding cty.Value
func MarshalResource(resource string, data *cty.Value) ([]byte, error) {
	s, err := GetProviderResourceSchema()
	if err != nil {
		return nil, err
	}
	t, err := GetObjectTypeFromSchema(s[resource])
	if err != nil {
		return nil, err
	}
	return msgpack.Marshal(*data, t)
}

// GVRFromCtyObject extracts a canonical schema.GroupVersionResource out of the resource's
// metadata by checking it against the discovery API via a RESTMapper
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
	if err != nil {
		return r, err
	}
	return gvr, nil
}

// GVRFromCtyUnstructured extracts a canonical schema.GroupVersionResource out of the resource's
// metadata by checking it against the discovery API via a RESTMapper
func GVRFromCtyUnstructured(o *unstructured.Unstructured) (schema.GroupVersionResource, error) {
	r := schema.GroupVersionResource{}
	m, err := GetRestMapper()
	if err != nil {
		return r, err
	}
	apv := o.GetAPIVersion()
	kind := o.GetKind()
	gv, err := schema.ParseGroupVersion(apv)
	if err != nil {
		return r, err
	}
	gvr, err := m.ResourceFor(gv.WithResource(kind))
	if err != nil {
		return r, err
	}
	return gvr, nil
}

// GVKFromCtyObject extracts a canonical schema.GroupVersionKind out of the resource's
// metadata by checking it agaings the discovery API via a RESTMapper
func GVKFromCtyObject(o *cty.Value) (schema.GroupVersionKind, error) {
	r := schema.GroupVersionKind{}
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
	gvk, err := m.KindFor(gv.WithResource(kind))
	if err != nil {
		return r, err
	}
	return gvk, nil
}

// CtyObjectToUnstructured converts a Terraform specific cty.Object type manifest
// into a Kubernetes dynamic client specific unstructured object
func CtyObjectToUnstructured(in *cty.Value) (map[string]interface{}, error) {
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

// UnstructuredToCty converts a Kubernetes dynamic client specific unstructured object
// into a Terraform specific cty.Object type manifest
func UnstructuredToCty(in map[string]interface{}) (cty.Value, error) {
	jsonVal, err := json.Marshal(in)
	if err != nil {
		return cty.NilVal, errors.Wrapf(err, "unable to marshal value")
	}
	simple := &ctyjson.SimpleJSONValue{}
	err = simple.UnmarshalJSON(jsonVal)
	if err != nil {
		return cty.NilVal, errors.Wrapf(err, "unable to unmarshal to simple value")
	}
	return simple.Value, nil
}

// IsResourceNamespaced determines if a resource is namespaced or cluster-level
// by querying the Kubernetes discovery API
func IsResourceNamespaced(gvr schema.GroupVersionResource) (bool, error) {
	d, err := GetDiscoveryClient()
	if err != nil {
		return false, err
	}
	rl, err := d.ServerResourcesForGroupVersion(gvr.GroupVersion().String())
	if err != nil {
		return false, err
	}
	for _, r := range rl.APIResources {
		// Dlog.Printf("[IsResourceNamespaced] Resource:%s Name:%s Namespaced:%s\n", spew.Sdump(gvr.Resource), spew.Sdump(r.Name), spew.Sdump(r.Namespaced))
		if strings.HasPrefix(r.Name, gvr.Resource) && !strings.Contains(r.Name, "/") {
			return r.Namespaced, nil
		}
	}
	return false, fmt.Errorf("resource %s not found", gvr.String())
}

// OpenAPIPathFromGVK returns the ID used to retrieve a resource type definition
// from the OpenAPI spec document
func OpenAPIPathFromGVK(gvk schema.GroupVersionKind) (string, error) {
	var repo string = "api"
	g := gvk.Group
	if g == "" {
		g = "core"
	}
	switch g {
	case "meta":
		repo = "apimachinery.pkg.apis"
	case "apiextensions.k8s.io":
		repo = "apiextensions-apiserver.pkg.apis"
	case "apiregistration.k8s.io":
		repo = "kube-aggregator.pkg.apis"
	}
	// the ID string that Swagger / OpenAPI uses to identify the resource
	// e.g. "io.k8s.apimachinery.pkg.apis.meta.v1.ObjectMeta"
	return strings.Join([]string{"io", "k8s", repo, strings.Split(g, ".")[0], gvk.Version, gvk.Kind}, "."), nil
}

// DeepUnknownVal creates a value given an arbitrary type
// with a default value of UnknownVal for all it's primitives.
func DeepUnknownVal(ty cty.Type) cty.Value {
	switch {
	case ty.IsObjectType():
		atts := ty.AttributeTypes()
		vals := make(map[string]cty.Value, len(atts))
		for name, att := range atts {
			vals[name] = DeepUnknownVal(att)
		}
		return cty.ObjectVal(vals)
	case ty.IsTupleType():
		atts := ty.TupleElementTypes()
		elems := make([]cty.Value, len(atts))
		for i, att := range atts {
			elems[i] = DeepUnknownVal(att)
		}
		return cty.TupleVal(elems)
	default:
		return cty.UnknownVal(ty)
	}
}
