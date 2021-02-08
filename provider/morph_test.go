package provider

import (
	"math/big"
	"reflect"
	"testing"

	"github.com/davecgh/go-spew/spew"
	"github.com/google/go-cmp/cmp"
	"github.com/hashicorp/terraform-plugin-go/tfprotov5/tftypes"
)

func TestMorphValueToType(t *testing.T) {
	type sampleInType struct {
		V tftypes.Value
		T tftypes.Type
	}
	samples := map[string]struct {
		In  sampleInType
		Out tftypes.Value
	}{
		"string->string": {
			In: sampleInType{
				V: tftypes.NewValue(tftypes.String, "hello"),
				T: tftypes.String,
			},
			Out: tftypes.NewValue(tftypes.String, "hello"),
		},
		"string->number": {
			In: sampleInType{
				V: tftypes.NewValue(tftypes.String, "12.4"),
				T: tftypes.Number,
			},
			Out: tftypes.NewValue(tftypes.Number, new(big.Float).SetFloat64(12.4)),
		},
		"string->bool": {
			In: sampleInType{
				V: tftypes.NewValue(tftypes.String, "true"),
				T: tftypes.Bool,
			},
			Out: tftypes.NewValue(tftypes.Bool, true),
		},
		"number->number": {
			In: sampleInType{
				V: tftypes.NewValue(tftypes.Number, new(big.Float).SetFloat64(12.4)),
				T: tftypes.Number,
			},
			Out: tftypes.NewValue(tftypes.Number, new(big.Float).SetFloat64(12.4)),
		},
		"number->string": {
			In: sampleInType{
				V: tftypes.NewValue(tftypes.Number, new(big.Float).SetFloat64(12.4)),
				T: tftypes.String,
			},
			Out: tftypes.NewValue(tftypes.String, "12.4"),
		},
		"bool->bool": {
			In: sampleInType{
				V: tftypes.NewValue(tftypes.Bool, true),
				T: tftypes.Bool,
			},
			Out: tftypes.NewValue(tftypes.Bool, true),
		},
		"bool->string": {
			In: sampleInType{
				V: tftypes.NewValue(tftypes.Bool, true),
				T: tftypes.String,
			},
			Out: tftypes.NewValue(tftypes.String, "true"),
		},
		"list->list": {
			In: sampleInType{
				V: tftypes.NewValue(tftypes.List{ElementType: tftypes.String}, []tftypes.Value{
					tftypes.NewValue(tftypes.String, "foo"),
					tftypes.NewValue(tftypes.String, "bar"),
					tftypes.NewValue(tftypes.String, "baz"),
				}),
				T: tftypes.List{ElementType: tftypes.String},
			},
			Out: tftypes.NewValue(tftypes.List{ElementType: tftypes.String}, []tftypes.Value{
				tftypes.NewValue(tftypes.String, "foo"),
				tftypes.NewValue(tftypes.String, "bar"),
				tftypes.NewValue(tftypes.String, "baz"),
			}),
		},
		"list->tuple": {
			In: sampleInType{
				V: tftypes.NewValue(tftypes.List{ElementType: tftypes.String}, []tftypes.Value{
					tftypes.NewValue(tftypes.String, "foo"),
					tftypes.NewValue(tftypes.String, "bar"),
					tftypes.NewValue(tftypes.String, "baz"),
				}),
				T: tftypes.Tuple{ElementTypes: []tftypes.Type{tftypes.String, tftypes.String, tftypes.String}},
			},
			Out: tftypes.NewValue(tftypes.Tuple{ElementTypes: []tftypes.Type{tftypes.String, tftypes.String, tftypes.String}}, []tftypes.Value{
				tftypes.NewValue(tftypes.String, "foo"),
				tftypes.NewValue(tftypes.String, "bar"),
				tftypes.NewValue(tftypes.String, "baz"),
			}),
		},
		"list->set": {
			In: sampleInType{
				V: tftypes.NewValue(tftypes.List{ElementType: tftypes.String}, []tftypes.Value{
					tftypes.NewValue(tftypes.String, "10"),
					tftypes.NewValue(tftypes.String, "11.9"),
					tftypes.NewValue(tftypes.String, "42"),
				}),
				T: tftypes.Set{ElementType: tftypes.Number},
			},
			Out: tftypes.NewValue(tftypes.Set{ElementType: tftypes.Number}, []tftypes.Value{
				tftypes.NewValue(tftypes.Number, new(big.Float).SetFloat64(10)),
				tftypes.NewValue(tftypes.Number, new(big.Float).SetFloat64(11.9)),
				tftypes.NewValue(tftypes.Number, new(big.Float).SetFloat64(42)),
			}),
		},
		"tuple->tuple": {
			In: sampleInType{
				V: tftypes.NewValue(tftypes.Tuple{ElementTypes: []tftypes.Type{tftypes.String, tftypes.String, tftypes.String}}, []tftypes.Value{
					tftypes.NewValue(tftypes.String, "foo"),
					tftypes.NewValue(tftypes.String, "bar"),
					tftypes.NewValue(tftypes.String, "baz"),
				}),
				T: tftypes.Tuple{ElementTypes: []tftypes.Type{tftypes.String, tftypes.String, tftypes.String}},
			},
			Out: tftypes.NewValue(tftypes.Tuple{ElementTypes: []tftypes.Type{tftypes.String, tftypes.String, tftypes.String}}, []tftypes.Value{
				tftypes.NewValue(tftypes.String, "foo"),
				tftypes.NewValue(tftypes.String, "bar"),
				tftypes.NewValue(tftypes.String, "baz"),
			}),
		},
		"tuple->list": {
			In: sampleInType{
				V: tftypes.NewValue(tftypes.Tuple{ElementTypes: []tftypes.Type{tftypes.String, tftypes.String, tftypes.String}}, []tftypes.Value{
					tftypes.NewValue(tftypes.String, "foo"),
					tftypes.NewValue(tftypes.String, "bar"),
					tftypes.NewValue(tftypes.String, "baz"),
				}),
				T: tftypes.List{ElementType: tftypes.String},
			},
			Out: tftypes.NewValue(tftypes.List{ElementType: tftypes.String}, []tftypes.Value{
				tftypes.NewValue(tftypes.String, "foo"),
				tftypes.NewValue(tftypes.String, "bar"),
				tftypes.NewValue(tftypes.String, "baz"),
			}),
		},
		"tuple->set": {
			In: sampleInType{
				V: tftypes.NewValue(tftypes.Tuple{ElementTypes: []tftypes.Type{tftypes.String, tftypes.String, tftypes.String}}, []tftypes.Value{
					tftypes.NewValue(tftypes.String, "foo"),
					tftypes.NewValue(tftypes.String, "bar"),
					tftypes.NewValue(tftypes.String, "baz"),
				}),
				T: tftypes.Set{ElementType: tftypes.String},
			},
			Out: tftypes.NewValue(tftypes.Set{ElementType: tftypes.String}, []tftypes.Value{
				tftypes.NewValue(tftypes.String, "foo"),
				tftypes.NewValue(tftypes.String, "bar"),
				tftypes.NewValue(tftypes.String, "baz"),
			}),
		},
		"set->tuple": {
			In: sampleInType{
				V: tftypes.NewValue(tftypes.Set{ElementType: tftypes.String}, []tftypes.Value{
					tftypes.NewValue(tftypes.String, "foo"),
					tftypes.NewValue(tftypes.String, "bar"),
					tftypes.NewValue(tftypes.String, "baz"),
				}),
				T: tftypes.Tuple{ElementTypes: []tftypes.Type{tftypes.String, tftypes.String, tftypes.String}},
			},
			Out: tftypes.NewValue(tftypes.Tuple{ElementTypes: []tftypes.Type{tftypes.String, tftypes.String, tftypes.String}}, []tftypes.Value{
				tftypes.NewValue(tftypes.String, "foo"),
				tftypes.NewValue(tftypes.String, "bar"),
				tftypes.NewValue(tftypes.String, "baz"),
			}),
		},
		"set->list": {
			In: sampleInType{
				V: tftypes.NewValue(tftypes.Set{ElementType: tftypes.String}, []tftypes.Value{
					tftypes.NewValue(tftypes.String, "foo"),
					tftypes.NewValue(tftypes.String, "bar"),
					tftypes.NewValue(tftypes.String, "baz"),
				}),
				T: tftypes.List{ElementType: tftypes.String},
			},
			Out: tftypes.NewValue(tftypes.List{ElementType: tftypes.String}, []tftypes.Value{
				tftypes.NewValue(tftypes.String, "foo"),
				tftypes.NewValue(tftypes.String, "bar"),
				tftypes.NewValue(tftypes.String, "baz"),
			}),
		},
		"map->object": {
			In: sampleInType{
				V: tftypes.NewValue(tftypes.Map{AttributeType: tftypes.String}, map[string]tftypes.Value{
					"one":   tftypes.NewValue(tftypes.String, "foo"),
					"two":   tftypes.NewValue(tftypes.String, "bar"),
					"three": tftypes.NewValue(tftypes.String, "baz"),
				}),
				T: tftypes.Object{AttributeTypes: map[string]tftypes.Type{
					"one":   tftypes.String,
					"two":   tftypes.String,
					"three": tftypes.String,
				}},
			},
			Out: tftypes.NewValue(tftypes.Object{AttributeTypes: map[string]tftypes.Type{
				"one":   tftypes.String,
				"two":   tftypes.String,
				"three": tftypes.String,
			}}, map[string]tftypes.Value{
				"one":   tftypes.NewValue(tftypes.String, "foo"),
				"two":   tftypes.NewValue(tftypes.String, "bar"),
				"three": tftypes.NewValue(tftypes.String, "baz"),
			}),
		},
		"object->map": {
			In: sampleInType{
				V: tftypes.NewValue(tftypes.Object{AttributeTypes: map[string]tftypes.Type{
					"one":   tftypes.String,
					"two":   tftypes.String,
					"three": tftypes.String,
				}}, map[string]tftypes.Value{
					"one":   tftypes.NewValue(tftypes.String, "foo"),
					"two":   tftypes.NewValue(tftypes.String, "bar"),
					"three": tftypes.NewValue(tftypes.String, "baz"),
				}),
				T: tftypes.Map{AttributeType: tftypes.String},
			},
			Out: tftypes.NewValue(tftypes.Map{AttributeType: tftypes.String}, map[string]tftypes.Value{
				"one":   tftypes.NewValue(tftypes.String, "foo"),
				"two":   tftypes.NewValue(tftypes.String, "bar"),
				"three": tftypes.NewValue(tftypes.String, "baz"),
			}),
		},
	}
	for n, s := range samples {
		t.Run(n, func(t *testing.T) {
			r, err := MorphValueToType(s.In.V, s.In.T, tftypes.AttributePath{})
			if err != nil {
				t.Logf("Failed type-morphing for sample '%s': %s", n, err)
				t.FailNow()
			}
			if !cmp.Equal(r, s.Out, cmp.Exporter(func(t reflect.Type) bool { return true })) {
				t.Logf("Result doesn't match expectation for sample '%s'", n)
				t.Logf("\t Sample:\t%s", spew.Sdump(s.In))
				t.Logf("\t Expected:\t%s", spew.Sdump(s.Out))
				t.Logf("\t Received:\t%s", spew.Sdump(r))
				t.Fail()
			}
		})
	}
}
