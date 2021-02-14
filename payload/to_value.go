package payload

import (
	"errors"
	"fmt"
	"math/big"
	"strconv"

	"github.com/hashicorp/terraform-plugin-go/tfprotov5/tftypes"
)

// ToTFValue converts a Kubernetes dynamic client unstructured object
// into a Terraform specific tftypes.Value type object
func ToTFValue(in interface{}, st tftypes.Type, at tftypes.AttributePath) (tftypes.Value, error) {
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
			return tftypes.NewValue(tftypes.String, in.(string)), nil
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
				iv, err = ToTFValue(v, st.(tftypes.List).ElementType, at.WithElementKeyInt(int64(k)))
			case st.Is(tftypes.Set{}):
				iv, err = ToTFValue(v, st.(tftypes.Set).ElementType, at.WithElementKeyInt(int64(k)))
			case st.Is(tftypes.Tuple{}):
				iv, err = ToTFValue(v, st.(tftypes.Tuple).ElementTypes[k], at.WithElementKeyInt(int64(k)))
			case st.Is(tftypes.DynamicPseudoType):
				iv, err = ToTFValue(v, tftypes.DynamicPseudoType, at.WithElementKeyInt(int64(k)))
			default:
				return tftypes.Value{}, fmt.Errorf("cannot convert unstructured list value: incompatible type: %s", st.String())
			}
			if err != nil {
				return tftypes.Value{}, fmt.Errorf("cannot convert unstructured list element value: %s", err)
			}
			il = append(il, iv)
		}
		if st.Is(tftypes.DynamicPseudoType) || st.(tftypes.List).ElementType.Is(tftypes.DynamicPseudoType) {
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
				return tftypes.Value{}, eap.NewErrorf("cannot convert unstructured map value: incompatible type: %s", st.String())
			}
			im[k], err = ToTFValue(v, kt, eap)
			if err != nil {
				return tftypes.Value{}, at.NewErrorf("cannot convert map element value: %s", err)
			}
		}
		switch {
		case st.Is(tftypes.DynamicPseudoType) || st.Is(tftypes.Object{}):
			oTypes := make(map[string]tftypes.Type)
			for k, v := range im {
				oTypes[k] = v.Type()
			}
			return tftypes.NewValue(tftypes.Object{AttributeTypes: oTypes}, im), nil
		case st.Is(tftypes.Map{}):
			et := tftypes.Type(nil)
			if len(im) > 0 {
				for k := range im {
					et = im[k].Type()
					break
				}
			} else {
				et = st.(tftypes.Map).AttributeType
			}
			return tftypes.NewValue(tftypes.Map{AttributeType: et}, im), nil
		}
	}
	return tftypes.Value{}, at.NewErrorf("[%s] cannot convert value of unknown type", at.String())
}
