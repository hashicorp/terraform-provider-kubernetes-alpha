package provider

import (
	"fmt"

	"github.com/hashicorp/terraform-plugin-go/tfprotov5/tftypes"
)

// TFValueDeepUnknown creates a value given an arbitrary type
// with a default value of Unknown for all its primitives.
func TFValueDeepUnknown(t tftypes.Type, v tftypes.Value) (tftypes.Value, error) {
	if t == nil {
		return tftypes.Value{}, fmt.Errorf("type cannot be nil")
	}
	switch {
	case t.Is(tftypes.Object{}):
		atts := t.(tftypes.Object).AttributeTypes
		vals := make(map[string]tftypes.Value, len(atts))
		err := v.As(&vals)
		if err != nil {
			return tftypes.Value{}, err
		}
		for name, att := range atts {
			nv, err := TFValueDeepUnknown(att, vals[name])
			if err != nil {
				return tftypes.Value{}, err
			}
			vals[name] = nv
		}
		return tftypes.NewValue(t, vals), nil
	case t.Is(tftypes.Map{}):
		vals := make(map[string]tftypes.Value, 0)
		err := v.As(&vals)
		if err != nil {
			return tftypes.Value{}, err
		}
		if len(vals) == 0 {
			return tftypes.NewValue(t, tftypes.UnknownValue), nil
		}
		for name, el := range vals {
			nv, err := TFValueDeepUnknown(t.(tftypes.Map).AttributeType, el)
			if err != nil {
				return tftypes.Value{}, err
			}
			vals[name] = nv
		}
		return tftypes.NewValue(t, vals), nil
	case t.Is(tftypes.Tuple{}):
		atts := t.(tftypes.Tuple).ElementTypes
		vals := make([]tftypes.Value, len(atts))
		err := v.As(&vals)
		if err != nil {
			return tftypes.Value{}, err
		}
		if len(vals) == 0 {
			return tftypes.NewValue(t, tftypes.UnknownValue), nil
		}
		for i, et := range atts {
			nv, err := TFValueDeepUnknown(et, vals[i])
			if err != nil {
				return tftypes.Value{}, err
			}
			vals[i] = nv
		}
		return tftypes.NewValue(t, vals), nil
	case t.Is(tftypes.List{}):
		vals := make([]tftypes.Value, 0)
		err := v.As(&vals)
		if err != nil {
			return tftypes.Value{}, err
		}
		if len(vals) == 0 {
			return tftypes.NewValue(t, tftypes.UnknownValue), nil
		}
		for i, el := range vals {
			nv, err := TFValueDeepUnknown(t.(tftypes.List).ElementType, el)
			if err != nil {
				return tftypes.Value{}, err
			}
			vals[i] = nv
		}
		return tftypes.NewValue(t, vals), nil
	default:
		if v.IsKnown() && !v.IsNull() {
			return v, nil
		}
		return tftypes.NewValue(t, tftypes.UnknownValue), nil
	}
}

// TFValueUnknownToNull replaces all unknown values in a deep structure with null
func TFValueUnknownToNull(v tftypes.Value) tftypes.Value {
	if !v.IsKnown() {
		return tftypes.NewValue(v.Type(), nil)
	}
	if v.IsNull() {
		return v
	}
	switch {
	case v.Type().Is(tftypes.List{}) || v.Type().Is(tftypes.Set{}) || v.Type().Is(tftypes.Tuple{}):
		tpel := make([]tftypes.Value, 0)
		v.As(&tpel)
		for i := range tpel {
			tpel[i] = TFValueUnknownToNull(tpel[i])
		}
		return tftypes.NewValue(v.Type(), tpel)
	case v.Type().Is(tftypes.Map{}) || v.Type().Is(tftypes.Object{}):
		mpel := make(map[string]tftypes.Value)
		v.As(&mpel)
		for k, ev := range mpel {
			mpel[k] = TFValueUnknownToNull(ev)
		}
		return tftypes.NewValue(v.Type(), mpel)
	}
	return v
}
