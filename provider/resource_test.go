package provider

import (
	"fmt"
	"reflect"
	"strings"
	"testing"

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
