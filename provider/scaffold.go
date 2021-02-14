package provider

import (
	"fmt"

	"github.com/hashicorp/terraform-plugin-go/tfprotov5/tftypes"
)

// TFValueDeepUnknown creates a value given an arbitrary type
// with a default value of Unknown for all its primitives.
func TFValueDeepUnknown(t tftypes.Type, v tftypes.Value, p tftypes.AttributePath) (tftypes.Value, error) {
	if t == nil {
		return tftypes.Value{}, fmt.Errorf("type cannot be nil")
	}
	if !v.IsKnown() {
		return v, nil
	}
	switch {
	case t.Is(tftypes.Object{}):
		atts := t.(tftypes.Object).AttributeTypes
		var vals map[string]tftypes.Value
		ovals := make(map[string]tftypes.Value, len(atts))
		err := v.As(&vals)
		if err != nil {
			return tftypes.Value{}, p.NewError(err)
		}
		for name, att := range atts {
			np := p.WithAttributeName(name)
			nv, err := TFValueDeepUnknown(att, vals[name], np)
			if err != nil {
				return tftypes.Value{}, np.NewError(err)
			}
			ovals[name] = nv
		}
		return tftypes.NewValue(t, ovals), nil
	case t.Is(tftypes.Map{}):
		if v.IsNull() {
			return tftypes.NewValue(t, tftypes.UnknownValue), nil
		}
		var vals map[string]tftypes.Value
		err := v.As(&vals)
		if err != nil {
			return tftypes.Value{}, p.NewError(err)
		}
		for name, el := range vals {
			np := p.WithElementKeyString(name)
			nv, err := TFValueDeepUnknown(t.(tftypes.Map).AttributeType, el, np)
			if err != nil {
				return tftypes.Value{}, np.NewError(err)
			}
			vals[name] = nv
		}
		return tftypes.NewValue(t, vals), nil
	case t.Is(tftypes.Tuple{}):
		if v.IsNull() {
			return tftypes.NewValue(t, tftypes.UnknownValue), nil
		}
		atts := t.(tftypes.Tuple).ElementTypes
		vals := make([]tftypes.Value, len(atts))
		err := v.As(&vals)
		if err != nil {
			return tftypes.Value{}, p.NewError(err)
		}
		for i, et := range atts {
			np := p.WithElementKeyInt(int64(i))
			nv, err := TFValueDeepUnknown(et, vals[i], np)
			if err != nil {
				return tftypes.Value{}, np.NewError(err)
			}
			vals[i] = nv
		}
		return tftypes.NewValue(t, vals), nil
	case t.Is(tftypes.List{}) || t.Is(tftypes.Set{}):
		if v.IsNull() {
			return tftypes.NewValue(t, tftypes.UnknownValue), nil
		}
		vals := make([]tftypes.Value, 0)
		err := v.As(&vals)
		if err != nil {
			return tftypes.Value{}, p.NewError(err)
		}
		var elt tftypes.Type
		switch {
		case t.Is(tftypes.List{}):
			elt = t.(tftypes.List).ElementType
		case t.Is(tftypes.Set{}):
			elt = t.(tftypes.Set).ElementType
		}
		for i, el := range vals {
			np := p.WithElementKeyInt(int64(i))
			nv, err := TFValueDeepUnknown(elt, el, np)
			if err != nil {
				return tftypes.Value{}, np.NewError(err)
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
