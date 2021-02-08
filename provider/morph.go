package provider

import (
	"math/big"
	"strconv"

	"github.com/hashicorp/terraform-plugin-go/tfprotov5/tftypes"
)

// MorphValueToType transforms a value along a new type and returns a new value conforming to the given type
func MorphValueToType(v tftypes.Value, t tftypes.Type, p tftypes.AttributePath) (tftypes.Value, error) {
	if t == nil {
		return tftypes.Value{}, p.NewErrorf("type is nil")
	}
	if v.IsNull() {
		return v, nil
	}
	if !v.IsKnown() {
		return v, p.NewErrorf("cannot morph value that isn't fully known")
	}
	switch {
	case v.Type().Is(tftypes.String):
		return morphStringToType(v, t, p)
	case v.Type().Is(tftypes.Number):
		return morphNumberToType(v, t, p)
	case v.Type().Is(tftypes.Bool):
		return morphBoolToType(v, t, p)
	case v.Type().Is(tftypes.DynamicPseudoType):
		return v, nil
	case v.Type().Is(tftypes.List{}):
		return morphListToType(v, t, p)
	case v.Type().Is(tftypes.Tuple{}):
		return morphTupleIntoType(v, t, p)
	case v.Type().Is(tftypes.Set{}):
		return morphSetToType(v, t, p)
	case v.Type().Is(tftypes.Map{}):
		return morphMapToType(v, t, p)
	case v.Type().Is(tftypes.Object{}):
		return morphObjectToType(v, t, p)
	}
	return tftypes.Value{}, p.NewErrorf("unsupported morph from value: %v", v)
}

func morphBoolToType(v tftypes.Value, t tftypes.Type, p tftypes.AttributePath) (tftypes.Value, error) {
	if t.Is(tftypes.Bool) {
		return v, nil
	}
	var bnat bool
	err := v.As(&bnat)
	if err != nil {
		return tftypes.Value{}, p.NewErrorf("failed to morph boolean value: %v", err)
	}
	switch {
	case t.Is(tftypes.String):
		return tftypes.NewValue(t, strconv.FormatBool(bnat)), nil
	case t.Is(tftypes.DynamicPseudoType):
		return v, nil
	}
	return tftypes.Value{}, p.NewErrorf("unsupported morph of bool value into type: %s", t.String())
}

func morphNumberToType(v tftypes.Value, t tftypes.Type, p tftypes.AttributePath) (tftypes.Value, error) {
	if t.Is(tftypes.Number) {
		return v, nil
	}
	var vnat big.Float
	err := v.As(&vnat)
	if err != nil {
		return tftypes.Value{}, p.NewErrorf("failed to morph number value: %v", err)
	}
	switch {
	case t.Is(tftypes.String):
		return tftypes.NewValue(t, vnat.String()), nil
	case t.Is(tftypes.DynamicPseudoType):
		return v, nil

	}
	return tftypes.Value{}, p.NewErrorf("unsupported morph of number value into type: %s", t.String())
}

func morphStringToType(v tftypes.Value, t tftypes.Type, p tftypes.AttributePath) (tftypes.Value, error) {
	if t.Is(tftypes.String) {
		return v, nil
	}
	var vnat string
	err := v.As(&vnat)
	if err != nil {
		return tftypes.Value{}, p.NewErrorf("failed to morph string value: %v", err)
	}
	switch {
	case t.Is(tftypes.Number):
		fv, err := strconv.ParseFloat(vnat, 64)
		if err != nil {
			return tftypes.Value{}, p.NewErrorf("failed to morph string value to tftypes.Number: %v", err)
		}
		nv := new(big.Float).SetFloat64(fv)
		return tftypes.NewValue(t, nv), nil
	case t.Is(tftypes.Bool):
		bv, err := strconv.ParseBool(vnat)
		if err != nil {
			return tftypes.Value{}, p.NewErrorf("failed to morph string value: %v", err)
		}
		return tftypes.NewValue(t, bv), nil
	case t.Is(tftypes.DynamicPseudoType):
		return v, nil
	}
	return tftypes.Value{}, p.NewErrorf("unsupported morph of string value into type: %s", t.String())
}

func morphListToType(v tftypes.Value, t tftypes.Type, p tftypes.AttributePath) (tftypes.Value, error) {
	if t.Is(tftypes.List{}) {
		return v, nil
	}
	var lvals []tftypes.Value
	err := v.As(&lvals)
	if err != nil {
		return tftypes.Value{}, p.NewErrorf("failed to morph list value: %v", err)
	}
	switch {
	case t.Is(tftypes.Tuple{}):
		if len(t.(tftypes.Tuple).ElementTypes) != len(lvals) {
			return tftypes.Value{}, p.NewErrorf("failed to morph list into tuple (length mismatch)")
		}
		var tvals []tftypes.Value = make([]tftypes.Value, len(lvals))
		for i, v := range lvals {
			p.WithElementKeyInt(int64(i))
			nv, err := MorphValueToType(v, t.(tftypes.Tuple).ElementTypes[i], p)
			if err != nil {
				return tftypes.Value{}, p.NewErrorf("failed to morph list element into tuple element: %v", err)
			}
			tvals[i] = nv
			p.WithoutLastStep()
		}
		return tftypes.NewValue(t, tvals), nil
	case t.Is(tftypes.Set{}):
		var svals []tftypes.Value = make([]tftypes.Value, len(lvals))
		for i, v := range lvals {
			p.WithElementKeyInt(int64(i))
			nv, err := MorphValueToType(v, t.(tftypes.Set).ElementType, p)
			if err != nil {
				return tftypes.Value{}, p.NewErrorf("failed to morph list element into set element: %v", err)
			}
			svals[i] = nv
			p.WithoutLastStep()
		}
		return tftypes.NewValue(t, svals), nil
	case t.Is(tftypes.DynamicPseudoType):
		return v, nil
	}
	return tftypes.Value{}, p.NewErrorf("unsupported morph of list value into type: %s", t.String())
}

func morphTupleIntoType(v tftypes.Value, t tftypes.Type, p tftypes.AttributePath) (tftypes.Value, error) {
	if t.Is(tftypes.Tuple{}) {
		return v, nil
	}
	var tvals []tftypes.Value
	err := v.As(&tvals)
	if err != nil {
		return tftypes.Value{}, p.NewErrorf("failed to morph tuple value: %v", err)
	}
	switch {
	case t.Is(tftypes.List{}):
		var lvals []tftypes.Value = make([]tftypes.Value, len(tvals))
		for i, v := range tvals {
			p.WithElementKeyInt(int64(i))
			nv, err := MorphValueToType(v, t.(tftypes.List).ElementType, p)
			if err != nil {
				return tftypes.Value{}, p.NewErrorf("failed to morph tuple element into list element: %v", err)
			}
			lvals[i] = nv
			p.WithoutLastStep()
		}
		return tftypes.NewValue(t, lvals), nil
	case t.Is(tftypes.Set{}):
		var svals []tftypes.Value = make([]tftypes.Value, len(tvals))
		for i, v := range tvals {
			p.WithElementKeyInt(int64(i))
			nv, err := MorphValueToType(v, t.(tftypes.Set).ElementType, p)
			if err != nil {
				return tftypes.Value{}, p.NewErrorf("failed to morph tuple element into set element: %v", err)
			}
			svals[i] = nv
			p.WithoutLastStep()
		}
		return tftypes.NewValue(t, svals), nil
	case t.Is(tftypes.DynamicPseudoType):
		return v, nil
	}
	return tftypes.Value{}, p.NewErrorf("unsupported morph of tuple value into type: %s", t.String())
}

func morphSetToType(v tftypes.Value, t tftypes.Type, p tftypes.AttributePath) (tftypes.Value, error) {
	if t.Is(tftypes.Set{}) {
		return v, nil
	}
	var svals []tftypes.Value
	err := v.As(&svals)
	if err != nil {
		return tftypes.Value{}, p.NewErrorf("failed to morph set value: %v", err)
	}
	switch {
	case t.Is(tftypes.List{}):
		var lvals []tftypes.Value = make([]tftypes.Value, len(svals))
		for i, v := range svals {
			p.WithElementKeyInt(int64(i))
			nv, err := MorphValueToType(v, t.(tftypes.List).ElementType, p)
			if err != nil {
				return tftypes.Value{}, p.NewErrorf("failed to morph set` element into list element: %v", err)
			}
			lvals[i] = nv
			p.WithoutLastStep()
		}
		return tftypes.NewValue(t, lvals), nil
	case t.Is(tftypes.Tuple{}):
		if len(t.(tftypes.Tuple).ElementTypes) != len(svals) {
			return tftypes.Value{}, p.NewErrorf("failed to morph list into tuple (length mismatch)")
		}
		var tvals []tftypes.Value = make([]tftypes.Value, len(svals))
		for i, v := range svals {
			p.WithElementKeyInt(int64(i))
			nv, err := MorphValueToType(v, t.(tftypes.Tuple).ElementTypes[i], tftypes.AttributePath{})
			if err != nil {
				return tftypes.Value{}, p.NewErrorf("failed to morph list element into tuple element: %v", err)
			}
			tvals[i] = nv
			p.WithoutLastStep()
		}
		return tftypes.NewValue(t, tvals), nil
	case t.Is(tftypes.DynamicPseudoType):
		return v, nil
	}
	return tftypes.Value{}, p.NewErrorf("unsupported morph of set value into type: %s", t.String())
}

func morphMapToType(v tftypes.Value, t tftypes.Type, p tftypes.AttributePath) (tftypes.Value, error) {
	if t.Is(tftypes.Map{}) {
		return v, nil
	}
	var mvals map[string]tftypes.Value
	err := v.As(&mvals)
	if err != nil {
		return tftypes.Value{}, p.NewErrorf("failed to morph map value: %v", err)
	}
	switch {
	case t.Is(tftypes.Object{}):
		var ovals map[string]tftypes.Value = make(map[string]tftypes.Value, len(mvals))
		for k, v := range mvals {
			p.WithElementKeyString(k)
			nv, err := MorphValueToType(v, t.(tftypes.Object).AttributeTypes[k], p)
			if err != nil {
				return tftypes.Value{}, p.NewErrorf("failed to morph map element into object element: %v", err)
			}
			ovals[k] = nv
			p.WithoutLastStep()
		}
		return tftypes.NewValue(t, ovals), nil
	case t.Is(tftypes.Map{}):
		var mvals map[string]tftypes.Value = make(map[string]tftypes.Value, len(mvals))
		for k, v := range mvals {
			p.WithElementKeyString(k)
			nv, err := MorphValueToType(v, t.(tftypes.Map).AttributeType, p)
			if err != nil {
				return tftypes.Value{}, p.NewErrorf("failed to morph object element into map element: %v", err)
			}
			mvals[k] = nv
			p.WithoutLastStep()
		}
		return tftypes.NewValue(t, mvals), nil
	case t.Is(tftypes.DynamicPseudoType):
		return v, nil
	}
	return tftypes.Value{}, p.NewErrorf("unsupported morph of map value into type: %s", t.String())
}

func morphObjectToType(v tftypes.Value, t tftypes.Type, p tftypes.AttributePath) (tftypes.Value, error) {
	var vals map[string]tftypes.Value
	err := v.As(&vals)
	if err != nil {
		return tftypes.Value{}, p.NewErrorf("failed to morph object value: %v", err)
	}
	switch {
	case t.Is(tftypes.Object{}):
		var ovals map[string]tftypes.Value = make(map[string]tftypes.Value, len(vals))
		for k, v := range vals {
			p.WithAttributeName(k)
			nv, err := MorphValueToType(v, t.(tftypes.Object).AttributeTypes[k], p)
			if err != nil {
				return tftypes.Value{}, p.NewErrorf("failed to morph object element into object element: %v", err)
			}
			ovals[k] = nv
			p.WithoutLastStep()
		}
		return tftypes.NewValue(t, ovals), nil
	case t.Is(tftypes.Map{}):
		var mvals map[string]tftypes.Value = make(map[string]tftypes.Value, len(vals))
		for k, v := range vals {
			p.WithElementKeyString(k)
			nv, err := MorphValueToType(v, t.(tftypes.Map).AttributeType, p)
			if err != nil {
				return tftypes.Value{}, p.NewErrorf("failed to morph object element into map element: %v", err)
			}
			mvals[k] = nv
			p.WithoutLastStep()
		}
		return tftypes.NewValue(t, mvals), nil
	case t.Is(tftypes.DynamicPseudoType):
		return v, nil
	}
	return tftypes.Value{}, p.NewErrorf("unsupported morph of object value into type: %s", t.String())
}
