package provider

import (
	"errors"
	"fmt"
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
