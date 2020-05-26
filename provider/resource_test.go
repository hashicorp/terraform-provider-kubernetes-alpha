package provider

import (
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/davecgh/go-spew/spew"
	"github.com/hashicorp/go-cty/cty"
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

func TestMorphManifestToOAPI(t *testing.T) {
	samples := []struct {
		in  cty.Value
		ty  cty.Type
		out cty.Value
	}{
		{
			in: cty.ObjectVal(map[string]cty.Value{
				"apiVersion": cty.StringVal("v1"),
				"kind":       cty.StringVal("ConfigMap"),
				"metadata": cty.ObjectVal(map[string]cty.Value{
					"name":      cty.StringVal("test-config"),
					"namespace": cty.StringVal("default"),
					"labels": cty.ObjectVal(map[string]cty.Value{
						"app":         cty.StringVal("test-app"),
						"environment": cty.StringVal("production"),
					}),
				}),
				"data": cty.ObjectVal(map[string]cty.Value{
					"foo": cty.StringVal("bar"),
				}),
			}),
			out: cty.ObjectVal(map[string]cty.Value{
				"apiVersion": cty.StringVal("v1"),
				"kind":       cty.StringVal("ConfigMap"),
				"metadata": cty.ObjectVal(map[string]cty.Value{
					"uid":               cty.NullVal(cty.String),
					"selfLink":          cty.NullVal(cty.String),
					"clusterName":       cty.NullVal(cty.String),
					"annotations":       cty.NullVal(cty.Map(cty.String)),
					"name":              cty.StringVal("test-config"),
					"resourceVersion":   cty.NullVal(cty.String),
					"creationTimestamp": cty.NullVal(cty.String),
					"generation":        cty.NullVal(cty.String),
					"ownerReferences": cty.NullVal(cty.List(cty.Object(map[string]cty.Type{
						"uid":                cty.String,
						"apiVersion":         cty.String,
						"blockOwnerDeletion": cty.Bool,
						"controller":         cty.Bool,
						"kind":               cty.String,
						"name":               cty.String,
					}))),
					"labels": cty.MapVal(map[string]cty.Value{
						"app":         cty.StringVal("test-app"),
						"environment": cty.StringVal("production"),
					}),
					"deletionGracePeriodSeconds": cty.NullVal(cty.Number),
					"generateName":               cty.NullVal(cty.String),
					"managedFields": cty.NullVal(cty.List(cty.Object(map[string]cty.Type{
						"apiVersion": cty.String,
						"fieldsType": cty.String,
						"fieldsV1":   cty.DynamicPseudoType,
						"manager":    cty.String,
						"operation":  cty.String,
						"time":       cty.String,
					}))),
					"finalizers":        cty.NullVal(cty.List(cty.String)),
					"deletionTimestamp": cty.NullVal(cty.String),

					"namespace": cty.StringVal("default"),
				}),
				"data": cty.MapVal(map[string]cty.Value{
					"foo": cty.StringVal("bar"),
				}),
				"binaryData": cty.NullVal(cty.Map(cty.String)),
			}),
			ty: cty.Object(map[string]cty.Type{
				"apiVersion": cty.String,
				"kind":       cty.String,
				"binaryData": cty.Map(cty.String),
				"data":       cty.Map(cty.String),
				"metadata": cty.Object(map[string]cty.Type{
					"uid":               cty.String,
					"selfLink":          cty.String,
					"clusterName":       cty.String,
					"annotations":       cty.Map(cty.String),
					"name":              cty.String,
					"resourceVersion":   cty.String,
					"creationTimestamp": cty.String,
					"generation":        cty.String,
					"ownerReferences": cty.List(cty.Object(map[string]cty.Type{
						"uid":                cty.String,
						"apiVersion":         cty.String,
						"blockOwnerDeletion": cty.Bool,
						"controller":         cty.Bool,
						"kind":               cty.String,
						"name":               cty.String,
					})),
					"labels":                     cty.Map(cty.String),
					"deletionGracePeriodSeconds": cty.Number,
					"generateName":               cty.String,
					"managedFields": cty.List(cty.Object(map[string]cty.Type{
						"apiVersion": cty.String,
						"fieldsType": cty.String,
						"fieldsV1":   cty.DynamicPseudoType,
						"manager":    cty.String,
						"operation":  cty.String,
						"time":       cty.String,
					})),
					"finalizers":        cty.List(cty.String),
					"deletionTimestamp": cty.String,
					"namespace":         cty.String,
				}),
			}),
		},
		{
			in: cty.ObjectVal(map[string]cty.Value{
				"apiVersion": cty.StringVal("rbac.authorization.k8s.io/v1"),
				"kind":       cty.StringVal("Role"),
				"metadata": cty.ObjectVal(map[string]cty.Value{
					"name":      cty.StringVal("test-role"),
					"namespace": cty.StringVal("default"),
					"labels": cty.ObjectVal(map[string]cty.Value{
						"app":         cty.StringVal("test-app"),
						"environment": cty.StringVal("production"),
					}),
				}),
				"rules": cty.TupleVal([]cty.Value{
					cty.ObjectVal(map[string]cty.Value{
						"apiGroups": cty.TupleVal([]cty.Value{
							cty.StringVal(""),
						}),
						"resources": cty.TupleVal([]cty.Value{
							cty.StringVal("pods"),
						}),
						"verbs": cty.TupleVal([]cty.Value{
							cty.StringVal("get"),
							cty.StringVal("list"),
							cty.StringVal("watch"),
						}),
					}),
				}),
			}),
			out: cty.ObjectVal(map[string]cty.Value{
				"apiVersion": cty.StringVal("rbac.authorization.k8s.io/v1"),
				"kind":       cty.StringVal("Role"),
				"metadata": cty.ObjectVal(map[string]cty.Value{
					"uid":               cty.NullVal(cty.String),
					"selfLink":          cty.NullVal(cty.String),
					"clusterName":       cty.NullVal(cty.String),
					"annotations":       cty.NullVal(cty.Map(cty.String)),
					"resourceVersion":   cty.NullVal(cty.String),
					"creationTimestamp": cty.NullVal(cty.String),
					"generation":        cty.NullVal(cty.String),
					"ownerReferences": cty.NullVal(cty.List(cty.Object(map[string]cty.Type{
						"uid":                cty.String,
						"apiVersion":         cty.String,
						"blockOwnerDeletion": cty.Bool,
						"controller":         cty.Bool,
						"kind":               cty.String,
						"name":               cty.String,
					}))),
					"deletionGracePeriodSeconds": cty.NullVal(cty.Number),
					"generateName":               cty.NullVal(cty.String),
					"managedFields": cty.NullVal(cty.List(cty.Object(map[string]cty.Type{
						"apiVersion": cty.String,
						"fieldsType": cty.String,
						"fieldsV1":   cty.DynamicPseudoType,
						"manager":    cty.String,
						"operation":  cty.String,
						"time":       cty.String,
					}))),
					"finalizers":        cty.NullVal(cty.List(cty.String)),
					"deletionTimestamp": cty.NullVal(cty.String),
					"labels": cty.MapVal(map[string]cty.Value{
						"app":         cty.StringVal("test-app"),
						"environment": cty.StringVal("production"),
					}),
					"name":      cty.StringVal("test-role"),
					"namespace": cty.StringVal("default"),
				}),
				"rules": cty.ListVal([]cty.Value{
					cty.ObjectVal(map[string]cty.Value{
						"apiGroups": cty.ListVal([]cty.Value{
							cty.StringVal(""),
						}),
						"resources": cty.ListVal([]cty.Value{
							cty.StringVal("pods"),
						}),
						"verbs": cty.ListVal([]cty.Value{
							cty.StringVal("get"),
							cty.StringVal("list"),
							cty.StringVal("watch"),
						}),
					}),
				}),
			}),
			ty: cty.Object(map[string]cty.Type{
				"apiVersion": cty.String,
				"kind":       cty.String,
				"metadata": cty.Object(map[string]cty.Type{
					"uid":               cty.String,
					"selfLink":          cty.String,
					"clusterName":       cty.String,
					"annotations":       cty.Map(cty.String),
					"name":              cty.String,
					"resourceVersion":   cty.String,
					"creationTimestamp": cty.String,
					"generation":        cty.String,
					"ownerReferences": cty.List(cty.Object(map[string]cty.Type{
						"uid":                cty.String,
						"apiVersion":         cty.String,
						"blockOwnerDeletion": cty.Bool,
						"controller":         cty.Bool,
						"kind":               cty.String,
						"name":               cty.String,
					})),
					"labels":                     cty.Map(cty.String),
					"deletionGracePeriodSeconds": cty.Number,
					"generateName":               cty.String,
					"managedFields": cty.List(cty.Object(map[string]cty.Type{
						"apiVersion": cty.String,
						"fieldsType": cty.String,
						"fieldsV1":   cty.DynamicPseudoType,
						"manager":    cty.String,
						"operation":  cty.String,
						"time":       cty.String,
					})),
					"finalizers":        cty.List(cty.String),
					"deletionTimestamp": cty.String,
					"namespace":         cty.String,
				}),
				"rules": cty.List(
					cty.Object(map[string]cty.Type{
						"apiGroups": cty.List(cty.String),
						"resources": cty.List(cty.String),
						"verbs":     cty.List(cty.String),
					})),
			}),
		},
	}
	for _, s := range samples {
		tt := strings.Join(
			[]string{s.in.GetAttr("apiVersion").AsString(), s.in.GetAttr("kind").AsString()},
			".",
		)
		t.Run(tt, func(t *testing.T) {
			o, err := morphManifestToOAPI(s.in, s.ty)
			no := cty.UnknownAsNull(o)
			if err != nil {
				t.Fatal(err)
			}
			if no.Equals(s.out).False() {
				t.Errorf("Morphed object does not match expectation.\nGOT: %s\nWANT: %s\n", spew.Sdump(o), spew.Sdump(s.out))
			}
		})
	}
}

func TestTypeForPath(t *testing.T) {
	samples := []struct {
		p      cty.Path
		in     cty.Type
		out    cty.Type
		expErr error
	}{
		{
			p:   cty.Path{},
			in:  cty.String,
			out: cty.String,
		},
		{
			in: cty.Object(map[string]cty.Type{
				"foo": cty.String,
				"bar": cty.Number,
			}),
			p:   cty.Path{}.GetAttr("foo"),
			out: cty.String,
		},
		{
			in: cty.Object(map[string]cty.Type{
				"foo": cty.Object(map[string]cty.Type{
					"bar": cty.Number,
				}),
			}),
			p:   cty.Path{}.GetAttr("foo").GetAttr("bar"),
			out: cty.Number,
		},
		{
			in: cty.Object(map[string]cty.Type{
				"foo": cty.List(cty.Object(map[string]cty.Type{
					"some":  cty.String,
					"other": cty.String,
				})),
				"bar": cty.Number,
			}),
			p:   cty.Path{}.GetAttr("foo").IndexInt(2).GetAttr("other"),
			out: cty.String,
		},
		{
			in: cty.Object(map[string]cty.Type{
				"foo": cty.String,
				"bar": cty.Number,
			}),
			p:   cty.Path{}.GetAttr("bar"),
			out: cty.Number,
		},
		{
			in: cty.Object(map[string]cty.Type{
				"metadata": cty.Object(map[string]cty.Type{
					"labels": cty.Map(cty.String),
				}),
			}),
			p:   cty.Path{}.GetAttr("metadata").GetAttr("labels").GetAttr("app"),
			out: cty.String,
		},
		{
			in: cty.Object(map[string]cty.Type{
				"metadata": cty.Object(map[string]cty.Type{
					"annotations": cty.Map(cty.String),
				}),
			}),
			p:      cty.Path{}.GetAttr("metadata").GetAttr("labels").GetAttr("app"),
			expErr: errors.New("type object has no attribute 'labels'"),
		},
		{
			in: cty.Object(map[string]cty.Type{
				"foo": cty.List(cty.String),
				"bar": cty.Number,
			}),
			p:   cty.Path{}.GetAttr("foo").IndexInt(2),
			out: cty.String,
		},
		{
			in: cty.Object(map[string]cty.Type{
				"foo": cty.List(cty.String),
				"bar": cty.Map(cty.String),
			}),
			p:   cty.Path{}.GetAttr("bar").IndexString("woo"),
			out: cty.String,
		},
		{
			in: cty.Object(map[string]cty.Type{
				"foo": cty.List(cty.String),
				"bar": cty.Map(cty.String),
			}),
			p:   cty.Path{}.GetAttr("bar"),
			out: cty.Map(cty.String),
		},
	}
	for i := range samples {
		var tt string
		if samples[i].expErr != nil {
			tt = "error:" + samples[i].expErr.Error()
		} else {
			tt = samples[i].out.FriendlyName()
		}
		t.Run(fmt.Sprintf("%s=>%s", DumpCtyPath(samples[i].p), tt),
			func(t *testing.T) {
				tr, err := typeForPath(samples[i].in, samples[i].p)
				if err != nil {
					expErr := samples[i].expErr
					if expErr != nil && strings.Contains(err.Error(), expErr.Error()) {
						return
					}
					t.Fatal(err)
				}
				if !samples[i].out.Equals(tr) {
					t.Errorf("path %s fails to apply to %s", samples[i].p, samples[i].in)
				}
			},
		)
	}
}
