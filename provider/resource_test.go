package provider

import (
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
				Group:   "apiextensions",
				Version: "v1beta1",
				Kind:    "CustomResourceColumnDefinition",
			},
			id: "io.k8s.apiextensions-apiserver.pkg.apis.apiextensions.v1beta1.CustomResourceColumnDefinition",
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
				Group:   "apiregistration",
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
			t.Fatalf("IDs don't match\n\tExpected: %s\n\tGot: %s", s.id, i)
		}
	}
}
