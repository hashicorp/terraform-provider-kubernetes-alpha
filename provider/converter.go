package provider

/* WIP: disabled

import (
	"fmt"
	"math/bits"

	"github.com/davecgh/go-spew/spew"
	"github.com/hashicorp/go-cty/cty"
)

// UnstructuredToCty converts a Kubernetes dynamic client specific unstructured object
// into a Terraform specific cty.Object type manifest
func UnstructuredToCty(in map[string]interface{}) (cty.Value, error) {
	cm := make(map[string]cty.Value, len(in))
	for k, v := range in {
		sv, err := UnstructuredToCtySingle(v)
		if err != nil {
			return cty.NilVal, err
		}
		cm[k] = sv
	}
	return cty.ObjectVal(cm), nil
}

// UnstructuredToCtyList converts a Kubernetes dynamic client specific unstructured list of objects
// into a Terraform specific cty.ListVal type manifest
func UnstructuredToCtyList(in []interface{}) (cty.Value, error) {
	nl := make([]cty.Value, len(in))
	for i, v := range in {
		sv, err := UnstructuredToCtySingle(v)
		if err != nil {
			return cty.NilVal, err
		}
		nl[i] = sv
	}
	return cty.ListVal(nl), nil
}

// UnstructuredToCtySingle converts a single Unstructured value to it's correspoding cty.Value
// Restrictions apply around numerical types - cty.Numerical is always represented as big.Float under the hood
func UnstructuredToCtySingle(in interface{}) (cty.Value, error) {
	switch tv := in.(type) {
	case map[string]interface{}:
		v, err := UnstructuredToCty(tv)
		if err != nil {
			return cty.NilVal, err
		}
		return v, nil
	case []interface{}:
		v, err := UnstructuredToCtyList(tv)
		if err != nil {
			return cty.NilVal, err
		}
		return v, nil
	case string:
		return cty.StringVal(tv), nil
	case int:
		return cty.NumberIntVal((int64)(tv)), nil
	case int64:
		return cty.NumberIntVal((int64)(tv)), nil
	case uint:
		if bits.Len64((uint64)(tv)) > 63 {
			return cty.NilVal, fmt.Errorf("value %#v doesn't fit into cty.Number", tv)
		}
		return cty.NumberIntVal((int64)(tv)), nil
	case uint64:
		if bits.Len64(tv) > 63 {
			return cty.NilVal, fmt.Errorf("value %#v doesn't fit into cty.Number", tv)
		}
		return cty.NumberIntVal((int64)(tv)), nil
	case float64:
		return cty.NumberFloatVal(tv), nil
	case float32:
		return cty.NumberFloatVal((float64)(tv)), nil
	case bool:
		return cty.BoolVal(tv), nil
	default:
		return cty.NilVal, fmt.Errorf("unknown type: %s", spew.Sdump(tv))
	}
}

// CtyObjectToUnstructured converts a Terraform specific cty.Object typed value
// into a Kubernetes dynamic client specific unstructured object
func CtyObjectToUnstructured(in *cty.Value) (map[string]interface{}, error) {
	if in == nil {
		return nil, fmt.Errorf("nil input")
	}
	if !in.Type().IsObjectType() {
		return nil, fmt.Errorf("not a cty.Object type (%s)", in.Type().GoString())
	}
	Dlog.Printf("[CtyObjectToUnstructured] %s", spew.Sdump(*in))
	m := make(map[string]interface{}, len(in.Type().AttributeTypes()))
	for it := in.ElementIterator(); it.Next(); {
		k, v := it.Element()
		switch {
		case v.Type().IsPrimitiveType():
			pv, err := ctyPrimitiveToUnstructured(&v)
			if err != nil {
				return nil, err
			}
			m[k.AsString()] = pv
		case v.Type().IsObjectType():
			ov, err := CtyObjectToUnstructured(&v)
			if err != nil {
				return nil, err
			}
			m[k.AsString()] = ov
		case v.Type().IsListType():
			lv, err := CtyListToUnstructured(&v)
			if err != nil {
				return nil, err
			}
			m[k.AsString()] = lv
		default:
			return nil, fmt.Errorf("object to Unstructured: no conversion defined for attribute '%s' from type %s",
				k.AsString(), v.Type().GoString())
		}
	}
	return m, nil
}

// CtyListToUnstructured converts a Terraform specific cty.List typed value
// into a Kubernetes dynamic client specific unstructured object
func CtyListToUnstructured(in *cty.Value) ([]interface{}, error) {
	if in == nil {
		return nil, fmt.Errorf("nil input")
	}
	if !in.Type().IsListType() {
		return nil, fmt.Errorf("not a cty.List type (%s)", in.Type().GoString())
	}
	Dlog.Printf("[CtyListToUnstructured] %s", spew.Sdump(*in))
	l := make([]interface{}, len(in.Type().AttributeTypes()))
	for it := in.ElementIterator(); it.Next(); {
		i, v := it.Element()
		switch {
		case v.Type().IsPrimitiveType():
			pv, err := ctyPrimitiveToUnstructured(&v)
			if err != nil {
				return nil, err
			}
			l = append(l, pv)
		case v.Type().IsObjectType():
			ov, err := CtyObjectToUnstructured(&v)
			if err != nil {
				return nil, err
			}
			l = append(l, ov)
		case v.Type().IsListType():
			lv, err := CtyListToUnstructured(&v)
			if err != nil {
				return nil, err
			}
			l = append(l, lv)
		case v.Type().IsTupleType():

		default:
			return nil, fmt.Errorf("list to Unstructured: no conversion defined for element [%s] from type %s",
				i.AsString(), in.Type().GoString())
		}
	}
	return l, nil
}

// CtyTupleToUnstructured converts a Terraform specific cty.Tuple typed value
// into a Kubernetes dynamic client specific unstructured object
func CtyTupleToUnstructured(in *cty.Value) ([]interface{}, error) {
	if in == nil {
		return nil, fmt.Errorf("nil input")
	}
	if !in.Type().IsTupleType() {
		return nil, fmt.Errorf("not a cty.List type (%s)", in.Type().GoString())
	}
}

// ctyPrimitiveToUnstructured converts a Terraform specific cty primitive typed value
// into a Kubernetes dynamic client specific unstructured object
func ctyPrimitiveToUnstructured(in *cty.Value) (interface{}, error) {
	if in == nil {
		return nil, fmt.Errorf("nil input")
	}
	if in.IsNull() {
		return nil, nil
	}
	if !in.IsKnown() {
		return nil, fmt.Errorf("cannot convert unknown value %#v", *in)
	}
	if !in.Type().IsPrimitiveType() {
		return nil, fmt.Errorf("cannot expand type %s as primitive", in.Type().GoString())
	}
	Dlog.Printf("[CtyPrimitiveToUnstructured] %s", spew.Sdump(*in))
	switch in.Type() {
	case cty.Bool:
		return in.True(), nil
	case cty.String:
		return in.AsString(), nil
	case cty.Number:
		bf := in.AsBigFloat()
		if bf.IsInt() {
			bi, _ := bf.Int64()
			return bi, nil
		}
		f, _ := bf.Float64()
		return f, nil
	default:
		return nil, fmt.Errorf("primitive to Unstructured: no conversion defined from type %s", in.Type().GoString())
	}
}
*/
