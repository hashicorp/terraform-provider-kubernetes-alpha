package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/hashicorp/go-cty/cty"
	"github.com/hashicorp/go-cty/cty/convert"
	ctyjson "github.com/hashicorp/go-cty/cty/json"
	"github.com/hashicorp/go-cty/cty/msgpack"
	"github.com/pkg/errors"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/dynamic"
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
		if !path.HasPrefix(prefix) || len(path) < len(prefix)+1 {
			return v, nil
		}
		var objpath cty.Path = path[len(prefix):]

		nv, err := objpath.Apply(*newobj)
		if err != nil {
			return v, nil
		}

		switch {
		case v.Type().IsPrimitiveType():
			if !v.Type().Equals(nv.Type()) {
				ncv, err := convert.Convert(nv, v.Type())
				if err != nil {
					return cty.NilVal, fmt.Errorf("failed to convert primitive type %s to %s", nv.Type().FriendlyName(), v.Type().FriendlyName())
				}
				return ncv, nil
			}
			return nv, nil

		case v.Type().IsObjectType():
			return v, nil

		case v.Type().IsListType():
			if !nv.IsNull() && nv.CanIterateElements() {
				nvraw := make([]cty.Value, 0, nv.LengthInt())
				for it := nv.ElementIterator(); it.Next(); {
					_, elm := it.Element()
					nvraw = append(nvraw, elm)
				}
				return cty.ListVal(nvraw), nil
			}
			return nv, nil

		case v.Type().IsMapType():
			if !nv.IsNull() && nv.CanIterateElements() {
				nvraw := make(map[string]cty.Value, nv.LengthInt())
				for it := nv.ElementIterator(); it.Next(); {
					idx, elm := it.Element()
					nvraw[idx.AsString()] = elm
				}
				return cty.MapVal(nvraw), nil
			}
			return nv, nil

		default:
			return cty.NilVal, fmt.Errorf("unimplemented type %s in transform", v.Type().FriendlyName())
		}
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

	jsonVal, err := ctyjson.Marshal(*in, in.Type())
	if err != nil {
		return nil, err
	}
	udata := map[string]interface{}{}
	err = json.Unmarshal(jsonVal, &udata)
	if err != nil {
		return nil, errors.Wrap(err, "unable to unmarshal json from cty value")
	}
	return udata, nil
}

// UnstructuredToCty converts a Kubernetes dynamic client specific unstructured object
// into a Terraform specific cty.Object type manifest
func UnstructuredToCty(in map[string]interface{}, t cty.Type) (cty.Value, error) {
	jsonVal, err := json.Marshal(in)
	if err != nil {
		return cty.NilVal, errors.Wrapf(err, "unable to marshal value")
	}

	v, err := ctyjson.Unmarshal(jsonVal, t)
	if err != nil {
		return cty.NilVal, errors.Wrapf(err, "unable to unmarshal json from unstructured value (%s - %s)", t.FriendlyName(), v.GoString())
	}

	return v, nil
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
// with a default value of NullVal for all it's primitives.
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
		vals := make([]cty.Value, len(atts))
		for i, t := range atts {
			vals[i] = DeepUnknownVal(t)
		}
		return cty.TupleVal(vals)
	default:
		return cty.UnknownVal(ty)
	}
}

func mapRemoveNulls(in map[string]interface{}) map[string]interface{} {
	for k, v := range in {
		switch tv := v.(type) {
		case []interface{}:
			in[k] = sliceRemoveNulls(tv)
		case map[string]interface{}:
			in[k] = mapRemoveNulls(tv)
		default:
			if v == nil {
				delete(in, k)
			}
		}
	}
	return in
}

func sliceRemoveNulls(in []interface{}) []interface{} {
	s := []interface{}{}
	for _, v := range in {
		switch tv := v.(type) {
		case []interface{}:
			s = append(s, sliceRemoveNulls(tv))
		case map[string]interface{}:
			s = append(s, mapRemoveNulls(tv))
		default:
			if v != nil {
				s = append(s, v)
			}
		}
	}
	return s
}

func resourceTypeFromOpenAPI(gvk schema.GroupVersionKind) (cty.Type, error) {
	id, err := OpenAPIPathFromGVK(gvk)
	if err != nil {
		return cty.NilType, errors.Wrap(err, "failed to determine resource type ID")
	}

	oapi, err := GetOAPIFoundry()
	if err != nil {
		return cty.NilType, errors.Wrap(err, "failed to get OpenAPI foundry")
	}

	tsch, err := oapi.GetTypeByID(id)
	if err != nil {
		return cty.NilType, errors.Wrapf(err, "failed to get resource type from OpenAPI (ID %s)", id)
	}

	// remove "status" attribute from resource type
	if tsch.IsObjectType() && tsch.HasAttribute("status") {
		atts := tsch.AttributeTypes()
		delete(atts, "status")
		tsch = cty.Object(atts)
	}

	return tsch, nil
}

func typeForPath(t cty.Type, p cty.Path) (cty.Type, error) {
	if len(p) == 0 {
		return t, nil
	}
	switch ts := p[0].(type) {
	case cty.IndexStep:
		if !t.IsListType() && !t.IsMapType() && !t.IsSetType() {
			return cty.NilType, fmt.Errorf("cannot use path step %s on type %s",
				p[0], t.GoString())
		}
		return typeForPath(t.ElementType(), p[1:])
	case cty.GetAttrStep:
		switch {
		case t.IsObjectType():
			if !t.HasAttribute(ts.Name) {
				return cty.NilType, fmt.Errorf("type %s has no attribute '%s'",
					t.FriendlyName(), ts.Name)
			}
			return typeForPath(t.AttributeType(ts.Name), p[1:])
		case t.IsMapType():
			return typeForPath(t.ElementType(), p[1:])
		default:
			return cty.NilType, fmt.Errorf("cannot use path step %s on type %s",
				p[0], t.GoString())
		}
	}
	return cty.NilType, errors.Errorf("cannot use path step %s on type %s",
		p[0], t.GoString())
}

func morphManifestToOAPI(m cty.Value, t cty.Type) (cty.Value, error) {
	nm, err := cty.Transform(m,
		func(p cty.Path, v cty.Value) (cty.Value, error) {
			ty, err := typeForPath(t, p)
			if err != nil {
				return cty.NilVal, errors.Wrapf(err, "failed to get type for path %s",
					DumpCtyPath(p))
			}
			switch {
			case ty.IsObjectType():
				if v.CanIterateElements() {
					vals := make(map[string]cty.Value, len(ty.AttributeTypes()))
					for idx, at := range ty.AttributeTypes() {
						if v.Type().HasAttribute(idx) {
							vals[idx] = v.GetAttr(idx)
						} else {
							vals[idx] = cty.UnknownVal(at)
						}
					}
					return cty.ObjectVal(vals), nil
				}
			case ty.IsMapType():
				if v.CanIterateElements() {
					vals := make(map[string]cty.Value, v.LengthInt())
					for it := v.ElementIterator(); it.Next(); {
						idx, ev := it.Element()
						vals[idx.AsString()] = ev
					}
					return cty.MapVal(vals), nil
				}
			case ty.IsListType():
				if v.CanIterateElements() {
					vals := make([]cty.Value, 0, v.LengthInt())
					for it := v.ElementIterator(); it.Next(); {
						_, ev := it.Element()
						vals = append(vals, ev)
					}
					return cty.ListVal(vals), nil
				}
			case ty.IsPrimitiveType():
				if !ty.Equals(v.Type()) {
					nv, err := convert.Convert(v, ty)
					if err != nil {
						return cty.NilVal, fmt.Errorf("failed to convert primitive '%s' to '%s'", v.Type().FriendlyName(), ty.FriendlyName())
					}
					return nv, nil
				}
			}
			return v, nil
		})
	if err != nil {
		return cty.NilVal, errors.Wrap(err, "failed to morph value")
	}
	return nm, nil
}

// PlanUpdateResource decides whether to off-load the change planning
// to the API server via a dry-run call or compute the changes locally
func PlanUpdateResource(ctx context.Context, in *cty.Value) (cty.Value, error) {
	s := GetProviderState()
	if s[SSPlanning].(bool) {
		return PlanUpdateResourceServerSide(ctx, in)
	}
	return PlanUpdateResourceLocal(ctx, in)
}

// PlanUpdateResourceLocal calculates the state for a new resource based on HCL manifest
func PlanUpdateResourceLocal(ctx context.Context, plan *cty.Value) (cty.Value, error) {
	m := plan.GetAttr("manifest")

	gvk, err := GVKFromCtyObject(&m)
	if err != nil {
		return cty.NilVal, fmt.Errorf("failed to determine resource GVR: %s", err)
	}

	tsch, err := resourceTypeFromOpenAPI(gvk)
	if err != nil {
		return cty.NilVal, fmt.Errorf("failed to determine resource type ID: %s", err)
	}

	// Transform the input manifest to adhere to the type modeled from the OpenAPI spec
	mobj, err := morphManifestToOAPI(m, tsch)

	var nc cty.Value

	if plan.GetAttr("object").IsNull() { // plan for Create
		nc, err = cty.Transform(*plan, ResourceBulkUpdateObjectAttr(&mobj))
		if err != nil {
			return cty.NilVal, err
		}
	} else { // plan for Update
		nnobj := cty.UnknownAsNull(mobj)
		nc, err = cty.Transform(*plan,
			ResourceDeepUpdateObjectAttr(cty.GetAttrPath("object"), &nnobj),
		)
		if err != nil {
			return cty.NilVal, err
		}
	}

	return nc, nil
}

// PlanUpdateResourceServerSide calculates the state for a new resource based on HCL manifest
func PlanUpdateResourceServerSide(ctx context.Context, p *cty.Value) (cty.Value, error) {
	m := p.GetAttr("manifest")

	co, err := CtyObjectToUnstructured(&m)
	if err != nil {
		return cty.NilVal, err
	}
	uo := unstructured.Unstructured{Object: co}

	gvr, err := GVRFromCtyUnstructured(&uo)
	if err != nil {
		return cty.NilVal, err
	}

	gvk, err := GVKFromCtyObject(&m)
	if err != nil {
		return cty.NilVal, fmt.Errorf("failed to determine resource GVR: %s", err)
	}

	tsch, err := resourceTypeFromOpenAPI(gvk)
	if err != nil {
		return cty.NilVal, fmt.Errorf("failed to determine resource type ID: %s", err)
	}

	ns, err := IsResourceNamespaced(gvr)
	if err != nil {
		return cty.NilVal, err
	}

	rnamespace := uo.GetNamespace()
	rname := uo.GetName()

	c, err := GetDynamicClient()
	if err != nil {
		return cty.NilVal, err
	}

	var r dynamic.ResourceInterface

	if ns {
		r = c.Resource(gvr).Namespace(rnamespace)
	} else {
		r = c.Resource(gvr)
	}

	js, err := uo.MarshalJSON()
	if err != nil {
		return cty.NilVal, err
	}

	ro, err := r.Patch(ctx,
		rname,
		types.ApplyPatchType,
		js,
		v1.PatchOptions{
			DryRun:       []string{v1.DryRunAll},
			FieldManager: "Terraform",
		})

	if err != nil {
		rn := types.NamespacedName{Namespace: rnamespace, Name: rname}.String()
		return cty.NilVal, fmt.Errorf("update dry-run for '%s' failed: %s", rn, err)
	}

	fo := FilterEphemeralFields(ro.Object)
	rc, err := UnstructuredToCty(fo, tsch)
	if err != nil {
		return cty.NilVal, err
	}
	np, err := cty.Transform(*p, ResourceBulkUpdateObjectAttr(&rc))
	if err != nil {
		return cty.NilVal, err
	}
	return np, nil
}

// FilterEphemeralFields removes certain fields from an API response object
// which would otherwise cause a false diff
func FilterEphemeralFields(in map[string]interface{}) map[string]interface{} {
	// Remove "status" attribute
	delete(in, "status")

	meta := in["metadata"].(map[string]interface{})

	// Remove "uid", "creationTimestamp", "resourceVersion" as
	// they change with most resource operations
	delete(meta, "uid")
	delete(meta, "creationTimestamp")
	delete(meta, "resourceVersion")
	delete(meta, "generation")
	delete(meta, "selfLink")

	// TODO: we should be filtering API responses based on the contents of 'managedFields'
	// and only retain the attributes for which the manager is Terraform
	delete(meta, "managedFields")

	return in
}
