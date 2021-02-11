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

// ResourceDeepMergeValue is a tftypes.Transform callback that sets each leaf node below the "object" attribute to a new cty.Value
func ResourceDeepMergeValue(aval, bval tftypes.Value) (tftypes.Value, error) {
	var err error

	if !aval.Type().Is(bval.Type()) {
		return tftypes.Value{}, errors.New("cannot merge values: incompatible types of A and B")
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
			return tftypes.Value{}, errors.New("cannot merge values: neither value is known")
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
			return tftypes.Value{}, fmt.Errorf("cannot merge values: cannot unpack object value A: %s", err)
		}

		err = bval.As(&Battributes)
		if err != nil {
			return tftypes.Value{}, fmt.Errorf("cannot merge values: cannot unpack object value B: %s", err)
		}

		for k := range Aattributes {
			if _, ok := Battributes[k]; ok {
				Rattributes[k], err = ResourceDeepMergeValue(Aattributes[k], Aattributes[k])
				if err != nil {
					return tftypes.Value{}, fmt.Errorf("cannot merge object elements: %s", err)
				}
				Rtype[k] = Rattributes[k].Type()
			}
		}
		return tftypes.NewValue(tftypes.Object{AttributeTypes: Rtype}, Rattributes), nil
	}
	return tftypes.Value{}, errors.New("cannot merge values: unknown combination")
}

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
	return schema.GroupVersionKind{}, errors.New("cannot select exact GV from REST mapper")
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
func TFValueToUnstructured(in tftypes.Value, ap tftypes.AttributePath) (interface{}, error) {
	var err error
	if !in.IsKnown() {
		return nil, ap.NewErrorf("[%s] cannot convert unknown value to Unstructured", ap.String())
	}
	if in.IsNull() {
		return nil, nil
	}
	if in.Type().Is(tftypes.DynamicPseudoType) {
		return nil, ap.NewErrorf("[%s] cannot convert dynamic value to Unstructured", ap.String())
	}
	switch {
	case in.Type().Is(tftypes.Bool):
		var bv bool
		err = in.As(&bv)
		if err != nil {
			return nil, ap.NewErrorf("[%s] cannot extract contents of attribute: %s", ap.String(), err)
		}
		return bv, nil
	case in.Type().Is(tftypes.Number):
		var nv big.Float
		err = in.As(&nv)
		if nv.IsInt() {
			inv, acc := nv.Int64()
			if acc != big.Exact {
				return nil, ap.NewErrorf("[%s] inexact integer approximation when converting number value at:", ap.String())
			}
			return inv, nil
		}
		fnv, acc := nv.Float64()
		if acc != big.Exact {
			return nil, ap.NewErrorf("[%s] inexact float approximation when converting number value", ap.String())
		}
		return fnv, err
	case in.Type().Is(tftypes.String):
		var sv string
		err = in.As(&sv)
		if err != nil {
			return nil, ap.NewErrorf("[%s] cannot extract contents of attribute: %s", ap.String(), err)
		}
		return sv, nil
	case in.Type().Is(tftypes.List{}) || in.Type().Is(tftypes.Tuple{}):
		var l []tftypes.Value
		var lv []interface{}
		err = in.As(&l)
		if err != nil {
			return nil, ap.NewErrorf("[%s] cannot extract contents of attribute: %s", ap.String(), err)
		}
		for k, le := range l {
			nextAp := ap.WithElementKeyInt(int64(k))
			ne, err := TFValueToUnstructured(le, nextAp)
			if err != nil {
				return nil, nextAp.NewErrorf("[%s] cannot convert list element to Unstructured: %s", nextAp.String(), err)
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
			return nil, ap.NewErrorf("[%s] cannot extract contents of attribute: %s", ap.String(), err)
		}
		for k, me := range m {
			var nextAp tftypes.AttributePath
			switch {
			case in.Type().Is(tftypes.Map{}):
				nextAp = ap.WithElementKeyString(k)
			case in.Type().Is(tftypes.Object{}):
				nextAp = ap.WithAttributeName(k)
			}
			ne, err := TFValueToUnstructured(me, nextAp)
			if err != nil {
				return nil, nextAp.NewErrorf("[%s]: cannot convert map element to Unstructured: %s", nextAp.String(), err.Error())
			}
			if ne != nil {
				mv[k] = ne
			}
		}
		if len(mv) == 0 {
			if strings.HasSuffix(ap.String(), `AttributeName("subresources").AttributeName("status")`) {
				// TODO: this is a horrible hack to work around the fact that `CustomResourceSubresourceStatus`
				// is specified as an empty object type in the OpenAPI spec. Because of that,
				return mv, nil
			}
			return nil, nil
		}
		return mv, nil
	default:
		return nil, ap.NewErrorf("[%s] cannot convert value of unknown type (%s)", ap.String(), in.Type().String())
	}
}

// UnstructuredToTFValue converts a Kubernetes dynamic client unstructured object
// into a Terraform specific tftypes.Value type object
func UnstructuredToTFValue(in interface{}, st tftypes.Type, at tftypes.AttributePath) (tftypes.Value, error) {
	if st == nil {
		return tftypes.Value{}, errors.New("type cannot be nil")
	}
	if in == nil {
		return tftypes.NewValue(st, nil), nil
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
		return tftypes.NewValue(tftypes.Number, new(big.Float).SetFloat64(in.(float64))), nil
	case []interface{}:
		var il []tftypes.Value
		for k, v := range in.([]interface{}) {
			var iv tftypes.Value
			switch {
			case st.Is(tftypes.List{}):
				iv, err = UnstructuredToTFValue(v, st.(tftypes.List).ElementType, at.WithElementKeyInt(int64(k)))
			case st.Is(tftypes.Set{}):
				iv, err = UnstructuredToTFValue(v, st.(tftypes.Set).ElementType, at.WithElementKeyInt(int64(k)))
			case st.Is(tftypes.Tuple{}):
				iv, err = UnstructuredToTFValue(v, st.(tftypes.Tuple).ElementTypes[k], at.WithElementKeyInt(int64(k)))
			case st.Is(tftypes.DynamicPseudoType):
				iv, err = UnstructuredToTFValue(v, tftypes.DynamicPseudoType, at.WithElementKeyInt(int64(k)))
			default:
				return tftypes.Value{}, fmt.Errorf("cannot convert unstructured list value: incompatible type: %s", st.String())
			}
			if err != nil {
				return tftypes.Value{}, fmt.Errorf("cannot convert unstructured list element value: %s", err)
			}
			il = append(il, iv)
		}
		if st.Is(tftypes.DynamicPseudoType) {
			tTypes := make([]tftypes.Type, len(il))
			for k := range il {
				tTypes[k] = il[k].Type()
			}
			return tftypes.NewValue(tftypes.Tuple{ElementTypes: tTypes}, il), nil
		}
		return tftypes.NewValue(st, il), nil
	case map[string]interface{}:
		im := make(map[string]tftypes.Value)
		for k, v := range in.(map[string]interface{}) {
			var kt tftypes.Type
			var eap tftypes.AttributePath
			switch {
			case st.Is(tftypes.Object{}):
				kt = st.(tftypes.Object).AttributeTypes[k]
				eap = at.WithAttributeName(k)
			case st.Is(tftypes.Map{}):
				kt = st.(tftypes.Map).AttributeType
				eap = at.WithElementKeyString(k)
			case st.Is(tftypes.DynamicPseudoType):
				kt = tftypes.DynamicPseudoType
				eap = eap.WithAttributeName(k)
			default:
				return tftypes.Value{}, at.NewErrorf("cannot convert unstructured map value: incompatible type: %s", st.String())
			}
			im[k], err = UnstructuredToTFValue(v, kt, eap)
			if err != nil {
				return tftypes.Value{}, at.NewErrorf("cannot convert map element value: %s", err)
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
	return tftypes.Value{}, errors.New("cannot convert value of unknown type")
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
		return nil, fmt.Errorf("cannot determine resource type ID: %s", err)
	}

	oapi, err := ps.GetOAPIFoundry()
	if err != nil {
		return nil, fmt.Errorf("cannot get OpenAPI foundry: %s", err)
	}

	tsch, err := oapi.GetTypeByID(id)
	if err != nil {
		return nil, fmt.Errorf("cannot get resource type from OpenAPI (ID %s): %s", id, err)
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
