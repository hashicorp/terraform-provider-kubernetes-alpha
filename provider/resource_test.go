package provider

import (
	"errors"
	"fmt"
	"math/big"
	"reflect"
	"strings"
	"testing"

	"github.com/davecgh/go-spew/spew"
	"github.com/google/go-cmp/cmp"
	"github.com/hashicorp/terraform-plugin-go/tfprotov5/tftypes"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

func TestOpenAPIPathFromGVR(t *testing.T) {
	samples := []struct {
		gvk schema.GroupVersionKind
		id  string
	}{
		{
			gvk: schema.GroupVersionKind{
				Group:   "apiextensions.k8s.io",
				Version: "v1beta1",
				Kind:    "CustomResourceDefinition",
			},
			id: "io.k8s.apiextensions-apiserver.pkg.apis.apiextensions.v1beta1.CustomResourceDefinition",
		},
		{
			gvk: schema.GroupVersionKind{
				Group:   "storage",
				Version: "v1beta1",
				Kind:    "StorageClass",
			},
			id: "io.k8s.api.storage.v1beta1.StorageClass",
		},
		{
			gvk: schema.GroupVersionKind{
				Group:   "apiregistration.k8s.io",
				Version: "v1",
				Kind:    "APIService",
			},
			id: "io.k8s.kube-aggregator.pkg.apis.apiregistration.v1.APIService",
		},
		{
			gvk: schema.GroupVersionKind{
				Group:   "meta",
				Version: "v1",
				Kind:    "ObjectMeta",
			},
			id: "io.k8s.apimachinery.pkg.apis.meta.v1.ObjectMeta",
		},
		{
			gvk: schema.GroupVersionKind{
				Group:   "",
				Version: "v1",
				Kind:    "Namespace",
			},
			id: "io.k8s.api.core.v1.Namespace",
		},
	}

	for _, s := range samples {
		i, err := OpenAPIPathFromGVK(s.gvk)
		if err != nil {
			t.Fatal(err)
		}
		if strings.Compare(i, s.id) != 0 {
			t.Fatalf("IDs don't match\n\tWant:\t%s\n\tGot:\t%s", s.id, i)
		}
	}
}

func TestRemoveNulls(t *testing.T) {
	samples := []struct {
		in  map[string]interface{}
		out map[string]interface{}
	}{
		{
			in: map[string]interface{}{
				"foo": nil,
			},
			out: map[string]interface{}{},
		},
		{
			in: map[string]interface{}{
				"foo": nil,
				"bar": "test",
			},
			out: map[string]interface{}{
				"bar": "test",
			},
		},
		{
			in: map[string]interface{}{
				"foo": nil,
				"bar": []interface{}{nil, "test"},
			},
			out: map[string]interface{}{
				"bar": []interface{}{"test"},
			},
		},
		{
			in: map[string]interface{}{
				"foo": nil,
				"bar": []interface{}{
					map[string]interface{}{
						"some":  nil,
						"other": "data",
					},
					"test",
				},
			},
			out: map[string]interface{}{
				"bar": []interface{}{
					map[string]interface{}{
						"other": "data",
					},
					"test",
				},
			},
		},
	}

	for i, s := range samples {
		t.Run(fmt.Sprintf("sample%d", i+1), func(t *testing.T) {
			o := mapRemoveNulls(s.in)
			if !reflect.DeepEqual(s.out, o) {
				t.Fatal("sample and output are not equal")
			}
		})
	}
}

func TestTFValueToUnstructured(t *testing.T) {
	samples := map[string]struct {
		In  tftypes.Value
		Out interface{}
	}{
		"string-primitive": {
			In:  tftypes.NewValue(tftypes.String, "hello"),
			Out: "hello",
		},
		"float-primitive": {
			In:  tftypes.NewValue(tftypes.Number, big.NewFloat(100.2)),
			Out: 100.2,
		},
		"boolean-primitive": {
			In:  tftypes.NewValue(tftypes.Bool, true),
			Out: true,
		},
		"list-of-strings": {
			In: tftypes.NewValue(tftypes.List{ElementType: tftypes.String}, []tftypes.Value{
				tftypes.NewValue(tftypes.String, "test1"),
				tftypes.NewValue(tftypes.String, "test2"),
			}),
			Out: []interface{}{"test1", "test2"},
		},
		"map-of-strings": {
			In: tftypes.NewValue(tftypes.Map{AttributeType: tftypes.String}, map[string]tftypes.Value{
				"foo": tftypes.NewValue(tftypes.String, "test1"),
				"bar": tftypes.NewValue(tftypes.String, "test2"),
			}),
			Out: map[string]interface{}{
				"foo": "test1",
				"bar": "test2",
			},
		},
		"object": {
			In: tftypes.NewValue(tftypes.Object{
				AttributeTypes: map[string]tftypes.Type{
					"foo":    tftypes.String,
					"buzz":   tftypes.Number,
					"fake":   tftypes.Bool,
					"others": tftypes.List{ElementType: tftypes.String},
				},
			}, map[string]tftypes.Value{
				"foo":  tftypes.NewValue(tftypes.String, "bar"),
				"buzz": tftypes.NewValue(tftypes.Number, new(big.Float).SetInt64(42)),
				"fake": tftypes.NewValue(tftypes.Bool, true),
				"others": tftypes.NewValue(tftypes.List{ElementType: tftypes.String}, []tftypes.Value{
					tftypes.NewValue(tftypes.String, "this"),
					tftypes.NewValue(tftypes.String, "that"),
				}),
			}),
			Out: map[string]interface{}{
				"foo":    "bar",
				"buzz":   int64(42),
				"fake":   true,
				"others": []interface{}{"this", "that"},
			},
		},
	}
	for n, s := range samples {
		t.Run(n, func(t *testing.T) {
			r, err := TFValueToUnstructured(s.In, tftypes.AttributePath{})
			if err != nil {
				t.Logf("Conversion failed for sample '%s': %s", n, err)
				t.FailNow()
			}
			if !reflect.DeepEqual(s.Out, r) {
				t.Logf("Result doesn't match expectation for sample '%s'", n)
				t.Logf("\t Sample:\t%s", spew.Sdump(s.In))
				t.Logf("\t Expected:\t%s", spew.Sdump(s.Out))
				t.Logf("\t Received:\t%s", spew.Sdump(r))
				t.Fail()
			}
		})
	}
}

func TestUnstructuredToTFValue(t *testing.T) {
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
			r, err := UnstructuredToTFValue(s.In.v, s.In.t, tftypes.AttributePath{})
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

func TestResourceDeepMergeValue(t *testing.T) {
	samples := map[string]struct {
		A   tftypes.Value
		B   tftypes.Value
		Out tftypes.Value
		E   error
	}{
		"unknown-unknown": {
			A:   tftypes.NewValue(tftypes.String, tftypes.UnknownValue),
			B:   tftypes.NewValue(tftypes.String, tftypes.UnknownValue),
			Out: tftypes.NewValue(tftypes.String, tftypes.UnknownValue),
			E:   nil,
		},
		"unknown-nil": {
			A:   tftypes.NewValue(tftypes.String, tftypes.UnknownValue),
			B:   tftypes.NewValue(tftypes.String, nil),
			Out: tftypes.NewValue(tftypes.String, nil),
			E:   nil,
		},
		"nil-unknown": {
			A:   tftypes.NewValue(tftypes.String, nil),
			B:   tftypes.NewValue(tftypes.String, tftypes.UnknownValue),
			Out: tftypes.NewValue(tftypes.String, nil),
			E:   nil,
		},
		"nil-string": {
			A:   tftypes.NewValue(tftypes.String, nil),
			B:   tftypes.NewValue(tftypes.String, "foobar"),
			Out: tftypes.NewValue(tftypes.String, "foobar"),
			E:   nil,
		},
		"unknown-string": {
			A:   tftypes.NewValue(tftypes.String, tftypes.UnknownValue),
			B:   tftypes.NewValue(tftypes.String, "foobar"),
			Out: tftypes.NewValue(tftypes.String, "foobar"),
			E:   nil,
		},
		"string-unknown": {
			A:   tftypes.NewValue(tftypes.String, "foobar"),
			B:   tftypes.NewValue(tftypes.String, tftypes.UnknownValue),
			Out: tftypes.NewValue(tftypes.String, "foobar"),
			E:   nil,
		},
		"string-nil": {
			A:   tftypes.NewValue(tftypes.String, "foobar"),
			B:   tftypes.NewValue(tftypes.String, nil),
			Out: tftypes.NewValue(tftypes.String, nil),
			E:   nil,
		},
		"string-list": {
			A:   tftypes.NewValue(tftypes.String, "foobar"),
			B:   tftypes.NewValue(tftypes.List{ElementType: tftypes.String}, nil),
			Out: tftypes.Value{},
			E:   errors.New("failed to merge values: incompatible types of A and B"),
		},
		"object-string": {
			A:   tftypes.NewValue(tftypes.Object{AttributeTypes: map[string]tftypes.Type{}}, nil),
			B:   tftypes.NewValue(tftypes.String, "foobar"),
			Out: tftypes.Value{},
			E:   errors.New("failed to merge values: incompatible types of A and B"),
		},
		"object-unknown": {
			A:   tftypes.NewValue(tftypes.Object{AttributeTypes: map[string]tftypes.Type{}}, nil),
			B:   tftypes.NewValue(tftypes.Object{AttributeTypes: map[string]tftypes.Type{}}, tftypes.UnknownValue),
			Out: tftypes.NewValue(tftypes.Object{AttributeTypes: map[string]tftypes.Type{}}, nil),
			E:   nil,
		},
	}
	for name, s := range samples {
		t.Run(name, func(t *testing.T) {
			r, err := ResourceDeepMergeValue(s.A, s.B)
			if err != nil {
				if s.E == nil {
					t.Logf("Unexpected error received for sample '%s': %s", name, err)
					t.FailNow()
				}
				if strings.Compare(err.Error(), s.E.Error()) != 0 {
					t.Logf("Error does not match expectation for sample'%s': %s", name, err)
					t.FailNow()
				}
			} else {
				if !cmp.Equal(r, s.Out, cmp.Exporter(func(reflect.Type) bool { return true })) {
					t.Logf("Result doesn't match expectation for sample '%s'", name)
					t.Logf("\t Sample A:\t%s", spew.Sdump(s.A))
					t.Logf("\t Sample B:\t%s", spew.Sdump(s.B))
					t.Logf("\t Expected:\t%s", spew.Sdump(s.Out))
					t.Logf("\t Received:\t%s", spew.Sdump(r))
					t.Fail()
				}
			}
		})
	}
}
