package provider

import (
	"errors"
	"fmt"
	"math/big"
	"strconv"
	"strings"

	"github.com/hashicorp/terraform-plugin-go/tfprotov5/tftypes"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

// ResourceDeepMergeValue is a cty.Transform callback that sets each leaf node below the "object" attribute to a new cty.Value
func ResourceDeepMergeValue(aval, bval tftypes.Value) (tftypes.Value, error) {
	var err error

	if !aval.Type().Is(bval.Type()) {
		return tftypes.Value{}, errors.New("failed to merge values: incompatible types of A and B")
	}

	switch {
	case aval.Type().Is(tftypes.String) || aval.Type().Is(tftypes.Number) || aval.Type().Is(tftypes.Bool):
		if bval.IsKnown() {
			return bval, nil
		}
		return aval, nil
	case aval.Type().Is(tftypes.Object{}):
		if !aval.IsKnown() {
			if bval.IsKnown() {
				return bval, nil
			}
			return tftypes.Value{}, errors.New("failed to merge values: neither value is known")
		}
		if !bval.IsKnown() {
			return aval, nil
		}

		var Aattributes map[string]tftypes.Value
		var Battributes map[string]tftypes.Value
		Rattributes := make(map[string]tftypes.Value) // result value attributes
		Rtype := make(map[string]tftypes.Type)

		err = aval.As(&Aattributes)
		if err != nil {
			return tftypes.Value{}, fmt.Errorf("failed to merge values: cannot unpack object value A: %s", err)
		}

		err = bval.As(&Battributes)
		if err != nil {
			return tftypes.Value{}, fmt.Errorf("failed to merge values: cannot unpack object value B: %s", err)
		}

		for k := range Aattributes {
			if _, ok := Battributes[k]; ok {
				Rattributes[k], err = ResourceDeepMergeValue(Aattributes[k], Aattributes[k])
				if err != nil {
					return tftypes.Value{}, fmt.Errorf("failed to merge object elements: %s", err)
				}
				Rtype[k] = Rattributes[k].Type()
			}
		}
		return tftypes.NewValue(tftypes.Object{AttributeTypes: Rtype}, Rattributes), nil
	}
	return tftypes.Value{}, errors.New("failed to merge values: unknown combination")
}

// GVRFromTftypesObject extracts a canonical schema.GroupVersionResource out of the resource's
// metadata by checking it against the discovery API via a RESTMapper
// func GVRFromTftypesObject(in *tftypes.Value, m meta.RESTMapper) (schema.GroupVersionResource, error) {
// 	r := schema.GroupVersionResource{}
// 	var obj map[string]tftypes.Value
// 	err := in.As(&obj)
// 	if err != nil {
// 		return r, err
// 	}
// 	var apv string
// 	var kind string
// 	err = obj["apiVersion"].As(&apv)
// 	if err != nil {
// 		return r, err
// 	}
// 	err = obj["kind"].As(&kind)
// 	if err != nil {
// 		return r, err
// 	}
// 	gv, err := schema.ParseGroupVersion(apv)
// 	if err != nil {
// 		return r, err
// 	}
// 	gvr, err := m.ResourceFor(gv.WithResource(kind))
// 	if err != nil {
// 		return r, err
// 	}
// 	return gvr, nil
// }

// GVRFromUnstructured extracts a canonical schema.GroupVersionResource out of the resource's
// metadata by checking it against the discovery API via a RESTMapper
func GVRFromUnstructured(o *unstructured.Unstructured, m meta.RESTMapper) (schema.GroupVersionResource, error) {
	apv := o.GetAPIVersion()
	kind := o.GetKind()
	gv, err := schema.ParseGroupVersion(apv)
	if err != nil {
		return schema.GroupVersionResource{}, err
	}
	mapping, err := m.RESTMapping(gv.WithKind(kind).GroupKind(), gv.Version)
	if err != nil {
		return schema.GroupVersionResource{}, err
	}
	return mapping.Resource, err
}

// GVKFromTftypesObject extracts a canonical schema.GroupVersionKind out of the resource's
// metadata by checking it agaings the discovery API via a RESTMapper
func GVKFromTftypesObject(in *tftypes.Value, m meta.RESTMapper) (schema.GroupVersionKind, error) {
	var obj map[string]tftypes.Value
	err := in.As(&obj)
	if err != nil {
		return schema.GroupVersionKind{}, err
	}
	var apv string
	var kind string
	err = obj["apiVersion"].As(&apv)
	if err != nil {
		return schema.GroupVersionKind{}, err
	}
	err = obj["kind"].As(&kind)
	if err != nil {
		return schema.GroupVersionKind{}, err
	}
	gv, err := schema.ParseGroupVersion(apv)
	if err != nil {
		return schema.GroupVersionKind{}, err
	}
	mappings, err := m.RESTMappings(gv.WithKind(kind).GroupKind())
	if err != nil {
		return schema.GroupVersionKind{}, err
	}
	for _, m := range mappings {
		if m.GroupVersionKind.GroupVersion().String() == apv {
			return m.GroupVersionKind, nil
		}
	}
	return schema.GroupVersionKind{}, errors.New("failed to select exact GV from REST mapper")
}

// IsResourceNamespaced determines if a resource is namespaced or cluster-level
// by querying the Kubernetes discovery API
func IsResourceNamespaced(gvk schema.GroupVersionKind, m meta.RESTMapper) (bool, error) {
	rm, err := m.RESTMapping(gvk.GroupKind(), gvk.Version)
	if err != nil {
		return false, err
	}
	if rm.Scope.Name() == meta.RESTScopeNameNamespace {
		return true, nil
	}
	return false, nil
}

// TFValueToUnstructured converts a Terraform specific tftypes.Value type object
// into a Kubernetes dynamic client specific unstructured object
func TFValueToUnstructured(in *tftypes.Value) (interface{}, error) {
	var err error
	if in.IsNull() || !in.IsKnown() {
		return nil, nil
	}
	if in.Type().Is(tftypes.DynamicPseudoType) {
		return nil, fmt.Errorf("cannot convert dynamic value to Unstructured")
	}
	switch {
	case in.Type().Is(tftypes.Bool):
		var bv bool
		err = in.As(&bv)
		return bv, err
	case in.Type().Is(tftypes.Number):
		var nv big.Float
		err = in.As(&nv)
		if nv.IsInt() {
			inv, acc := nv.Int64()
			if acc != big.Exact {
				return inv, fmt.Errorf("inexact integer approximation when converting number value")
			}
			return inv, err
		}
		fnv, acc := nv.Float64()
		if acc != big.Exact {
			return fnv, fmt.Errorf("inexact float approximation when converting number value")
		}
		return fnv, err
	case in.Type().Is(tftypes.String):
		var sv string
		err = in.As(&sv)
		return sv, err
	case in.Type().Is(tftypes.List{}) || in.Type().Is(tftypes.Tuple{}):
		var l []tftypes.Value
		var lv []interface{}
		err = in.As(&l)
		if err != nil {
			return lv, err
		}
		for k, le := range l {
			ne, err := TFValueToUnstructured(&le)
			if err != nil {
				return lv, fmt.Errorf("cannot convert list element %d to Unstructured: %s", k, err.Error())
			}
			if ne != nil {
				lv = append(lv, ne)
			}
		}
		if len(lv) == 0 {
			return nil, nil
		}
		return lv, nil
	case in.Type().Is(tftypes.Map{}) || in.Type().Is(tftypes.Object{}):
		m := make(map[string]tftypes.Value)
		mv := make(map[string]interface{})
		err = in.As(&m)
		if err != nil {
			return mv, err
		}
		for k, me := range m {
			ne, err := TFValueToUnstructured(&me)
			if err != nil {
				return mv, fmt.Errorf("cannot convert `map element %s to Unstructured: %s", k, err.Error())
			}
			if ne != nil {
				mv[k] = ne
			}
		}
		if len(mv) == 0 {
			return nil, nil
		}
		return mv, nil
	default:
		return nil, fmt.Errorf("cannot convert value of unknown type")
	}
}

// UnstructuredToTFValue converts a Kubernetes dynamic client unstructured object
// into a Terraform specific tftypes.Value type object
func UnstructuredToTFValue(in interface{}, st tftypes.Type, at tftypes.AttributePath) (tftypes.Value, error) {
	// var err error
	if in == nil {
		return tftypes.NewValue(tftypes.DynamicPseudoType, nil), nil
	}
	var err error
	switch in.(type) {
	case string:
		switch {
		case st.Is(tftypes.Number):
			num, err := strconv.Atoi(in.(string))
			if err != nil {
				return tftypes.Value{}, err
			}
			return tftypes.NewValue(tftypes.Number, num), nil
		default:
			return tftypes.NewValue(tftypes.String, in), nil
		}
	case bool:
		return tftypes.NewValue(tftypes.Bool, in), nil
	case int:
		return tftypes.NewValue(tftypes.Number, new(big.Float).SetInt64(int64(in.(int)))), nil
	case int64:
		return tftypes.NewValue(tftypes.Number, new(big.Float).SetInt64(in.(int64))), nil
	case int32:
		return tftypes.NewValue(tftypes.Number, new(big.Float).SetInt64(int64(in.(int32)))), nil
	case int16:
		return tftypes.NewValue(tftypes.Number, new(big.Float).SetInt64(int64(in.(int16)))), nil
	case float64:
		return tftypes.NewValue(tftypes.Number, &in), nil
	case []interface{}:
		var il []tftypes.Value
		for k, v := range in.([]interface{}) {
			var iv tftypes.Value
			switch {
			case st.Is(tftypes.List{}):
				iv, err = UnstructuredToTFValue(v, st.(tftypes.List).ElementType, at)
			case st.Is(tftypes.Set{}):
				iv, err = UnstructuredToTFValue(v, st.(tftypes.Set).ElementType, at)
			case st.Is(tftypes.Tuple{}):
				iv, err = UnstructuredToTFValue(v, st.(tftypes.Tuple).ElementTypes[k], at)
			case st.Is(tftypes.DynamicPseudoType):
				iv, err = UnstructuredToTFValue(v, tftypes.DynamicPseudoType, at)
			default:
				return tftypes.Value{}, fmt.Errorf("failed to convert unstructured list value: incompatible type: %s", st.String())
			}
			if err != nil {
				return tftypes.Value{}, fmt.Errorf("failed to convert unstructured list element value: %s", err)
			}
			il = append(il, iv)
		}
		if st.Is(tftypes.DynamicPseudoType) {
			var lType tftypes.Type = il[0].Type()
			for i := 1; i < len(il); i++ {
				if !lType.Is(il[i].Type()) {
					// attributes have differnent types - it's a tupple
					tTypes := make([]tftypes.Type, len(il))
					for k := range il {
						tTypes[k] = il[k].Type()
					}
					return tftypes.NewValue(tftypes.Tuple{ElementTypes: tTypes}, il), nil
				}
			}
			// all key types match  - it's a list
			return tftypes.NewValue(tftypes.List{ElementType: il[0].Type()}, il), nil
		}
		return tftypes.NewValue(st, il), nil
	case map[string]interface{}:
		im := make(map[string]tftypes.Value)
		for k, v := range in.(map[string]interface{}) {
			var kt tftypes.Type
			switch {
			case st.Is(tftypes.Object{}):
				kt = st.(tftypes.Object).AttributeTypes[k]
			case st.Is(tftypes.Map{}):
				kt = st.(tftypes.Map).AttributeType
			case st.Is(tftypes.DynamicPseudoType):
				kt = tftypes.DynamicPseudoType
			default:
				return tftypes.Value{}, fmt.Errorf("failed to convert unstructured map value: incompatible type: %s", st.String())
			}
			im[k], err = UnstructuredToTFValue(v, kt, at)
			if err != nil {
				return tftypes.Value{}, fmt.Errorf("failed to convert map element value: %s", err)
			}
		}
		if st.Is(tftypes.DynamicPseudoType) {
			oTypes := make(map[string]tftypes.Type)
			for k, v := range im {
				oTypes[k] = v.Type()
			}
			return tftypes.NewValue(tftypes.Object{AttributeTypes: oTypes}, im), nil
		}
		return tftypes.NewValue(st, im), nil
	}
	return tftypes.Value{}, at.NewErrorf("cannot convert value of unknown type")
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

// TFTypeFromOpenAPI generates a tftypes.Type representation of a Kubernetes resource
// designated by the supplied GroupVersionKind resource id
func (ps *RawProviderServer) TFTypeFromOpenAPI(gvk schema.GroupVersionKind) (tftypes.Type, error) {
	id, err := OpenAPIPathFromGVK(gvk)
	if err != nil {
		return tftypes.DynamicPseudoType, fmt.Errorf("failed to determine resource type ID: %s", err)
	}

	oapi, err := ps.GetOAPIFoundry()
	if err != nil {
		return tftypes.DynamicPseudoType, fmt.Errorf("failed to get OpenAPI foundry: %s", err)
	}

	tsch, err := oapi.GetTypeByID(id)
	if err != nil {
		return tftypes.DynamicPseudoType, fmt.Errorf("failed to get resource type from OpenAPI (ID %s): %s", id, err)
	}

	// remove "status" attribute from resource type
	if tsch.Is(tftypes.Object{}) {
		ot := tsch.(tftypes.Object)
		_, ok := ot.AttributeTypes["status"]
		if ok {
			atts := ot.AttributeTypes
			delete(atts, "status")
			tsch = tftypes.Object{AttributeTypes: atts}
		}
	}

	return tsch, nil
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

// TFTypeOfValue returns an equivalent type of an arbitrarily complex tftypes.Value
// by infering it using a breadth-first traversal of the input structure.
//
// NOTE:
// This solution is not ideal and was developed as a workaround for not being able to
// directly extract the type of a tftypes.Value due to it being private.
// This should be resolved when https://github.com/hashicorp/terraform-plugin-go/pull/58 merges
//
// TODO: We should remove this function after validating the solution from above PR.
func TFTypeOfValue(in tftypes.Value) (tftypes.Type, error) {
	if !in.IsKnown() {
		return tftypes.DynamicPseudoType, nil
	}
	var err error
	switch {
	case in.Type().Is(tftypes.Number):
		return tftypes.Number, nil
	case in.Type().Is(tftypes.String):
		return tftypes.String, nil
	case in.Type().Is(tftypes.Bool):
		return tftypes.Bool, nil
	case in.Type().Is(tftypes.DynamicPseudoType):
		return tftypes.DynamicPseudoType, nil
	case in.Type().Is(tftypes.Map{}):
		atm := make(map[string]tftypes.Value)
		in.As(&atm)
		var t tftypes.Type
		for _, v := range atm {
			t, err = TFTypeOfValue(v)
			if err != nil {
				return nil, err
			}
			break
		}
		return tftypes.Map{AttributeType: t}, nil
	case in.Type().Is(tftypes.Object{}):
		atm := make(map[string]tftypes.Value)
		tpm := make(map[string]tftypes.Type)
		in.As(&atm)
		for k, v := range atm {
			tpm[k], err = TFTypeOfValue(v)
			if err != nil {
				return nil, err
			}
		}
		return tftypes.Object{AttributeTypes: tpm}, nil
	case in.Type().Is(tftypes.List{}):
		var el []tftypes.Value
		in.As(&el)
		t, err := TFTypeOfValue(el[0])
		if err != nil {
			return nil, err
		}
		return tftypes.List{ElementType: t}, nil
	case in.Type().Is(tftypes.Set{}):
		var el []tftypes.Value
		in.As(&el)
		t, err := TFTypeOfValue(el[0])
		if err != nil {
			return nil, err
		}
		return tftypes.Set{ElementType: t}, err
	case in.Type().Is(tftypes.Tuple{}):
		var el []tftypes.Value
		var t []tftypes.Type
		in.As(&el)
		for i, v := range el {
			t[i], err = TFTypeOfValue(v)
			if err != nil {
				return nil, err
			}
		}
		return tftypes.Tuple{ElementTypes: t}, nil
	}
	return nil, fmt.Errorf("cannot determine type for value: %#v", in)
}

/*
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

*/
