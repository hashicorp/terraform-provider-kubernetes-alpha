package payload

import (
	"errors"
	"math/big"
	"reflect"
	"strings"
	"testing"

	"github.com/davecgh/go-spew/spew"
	"github.com/hashicorp/terraform-plugin-go/tfprotov5/tftypes"
)

func TestToTFValue(t *testing.T) {
	type sampleInType struct {
		v interface{}
		t tftypes.Type
	}
	samples := map[string]struct {
		In  sampleInType
		Out tftypes.Value
		Err error
	}{
		"string": {
			In:  sampleInType{v: "foobar", t: nil},
			Out: tftypes.NewValue(tftypes.String, "foobar"),
			Err: errors.New("type cannot be nil"),
		},
		"boolean": {
			In:  sampleInType{v: true, t: tftypes.Bool},
			Out: tftypes.NewValue(tftypes.Bool, true),
			Err: nil,
		},
		"integer": {
			In:  sampleInType{v: int64(100), t: tftypes.Number},
			Out: tftypes.NewValue(tftypes.Number, new(big.Float).SetInt64(100)),
			Err: nil,
		},
		"integer64": {
			In:  sampleInType{v: int64(0x100000000), t: tftypes.Number},
			Out: tftypes.NewValue(tftypes.Number, new(big.Float).SetInt64(0x100000000)),
			Err: nil,
		},
		"integer32": {
			In:  sampleInType{int32(0x01000000), tftypes.Number},
			Out: tftypes.NewValue(tftypes.Number, new(big.Float).SetInt64(0x01000000)),
			Err: nil,
		},
		"integer16": {
			In:  sampleInType{int16(0x0100), tftypes.Number},
			Out: tftypes.NewValue(tftypes.Number, new(big.Float).SetInt64(0x0100)),
			Err: nil,
		},
		"float64": {
			In:  sampleInType{float64(100.0), tftypes.Number},
			Out: tftypes.NewValue(tftypes.Number, new(big.Float).SetFloat64(100)),
			Err: nil,
		},
		"list": {
			In: sampleInType{[]interface{}{"test1", "test2"}, tftypes.List{ElementType: tftypes.String}},
			Out: tftypes.NewValue(tftypes.List{ElementType: tftypes.String}, []tftypes.Value{
				tftypes.NewValue(tftypes.String, "test1"),
				tftypes.NewValue(tftypes.String, "test2"),
			}),
			Err: nil,
		},
		"map": {
			In: sampleInType{
				v: map[string]interface{}{
					"foo": 18,
					"bar": "crawl",
				},
				t: tftypes.Map{AttributeType: tftypes.String},
			},
			Out: tftypes.NewValue(tftypes.Map{AttributeType: tftypes.String}, map[string]tftypes.Value{
				"foo": tftypes.NewValue(tftypes.Number, new(big.Float).SetInt64(18)),
				"bar": tftypes.NewValue(tftypes.String, "crawl"),
			}),
			Err: nil,
		},
		"complex-map": {
			In: sampleInType{
				v: map[string]interface{}{
					"foo": []interface{}{"test1", "test2"},
					"bar": map[string]interface{}{
						"count": 1,
						"image": "nginx/latest",
					},
					"refresh": true,
				},
				t: tftypes.Object{AttributeTypes: map[string]tftypes.Type{
					"foo": tftypes.List{ElementType: tftypes.String},
					"bar": tftypes.Object{AttributeTypes: map[string]tftypes.Type{
						"count": tftypes.Number,
						"image": tftypes.String,
					}},
					"refresh": tftypes.Bool,
				}},
			},
			Out: tftypes.NewValue(
				tftypes.Object{AttributeTypes: map[string]tftypes.Type{
					"foo": tftypes.List{ElementType: tftypes.String},
					"bar": tftypes.Object{AttributeTypes: map[string]tftypes.Type{
						"count": tftypes.Number,
						"image": tftypes.String,
					}},
					"refresh": tftypes.Bool,
				}},
				map[string]tftypes.Value{
					"foo": tftypes.NewValue(tftypes.List{ElementType: tftypes.String}, []tftypes.Value{
						tftypes.NewValue(tftypes.String, "test1"),
						tftypes.NewValue(tftypes.String, "test2"),
					}),
					"bar": tftypes.NewValue(
						tftypes.Object{AttributeTypes: map[string]tftypes.Type{
							"count": tftypes.Number,
							"image": tftypes.String,
						}},
						map[string]tftypes.Value{
							"count": tftypes.NewValue(tftypes.Number, new(big.Float).SetInt64(1)),
							"image": tftypes.NewValue(tftypes.String, "nginx/latest"),
						}),
					"refresh": tftypes.NewValue(tftypes.Bool, true),
				}),
		},
	}
	for name, s := range samples {
		t.Run(name, func(t *testing.T) {
			r, err := ToTFValue(s.In.v, s.In.t, tftypes.AttributePath{})
			if err != nil {
				if s.Err == nil {
					t.Logf("Unexpected error received for sample '%s': %s", name, err)
					t.FailNow()
				}
				if strings.Compare(err.Error(), s.Err.Error()) != 0 {
					t.Logf("Error does not match expectation for sample'%s': %s", name, err)
					t.FailNow()
				}
			} else {
				if !reflect.DeepEqual(s.Out, r) {
					t.Logf("Result doesn't match expectation for sample '%s'", name)
					t.Logf("\t Sample:\t%s", spew.Sdump(s.In))
					t.Logf("\t Expected:\t%s", spew.Sdump(s.Out))
					t.Logf("\t Received:\t%s", spew.Sdump(r))
					t.Fail()
				}
			}
		})
	}
}
