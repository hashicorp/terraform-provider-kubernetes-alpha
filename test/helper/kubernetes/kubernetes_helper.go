package kubernetes

import (
	"context"
	"math"
	"testing"
	"time"

	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// Helper is a Kubernetes dynamic client wrapped with a set of helper functions
// for making assertions about API resources
type Helper struct {
	client dynamic.Interface
}

// NewHelper initializes a new Kubernetes client
func NewHelper() *Helper {
	rules := clientcmd.NewDefaultClientConfigLoadingRules()
	overrides := &clientcmd.ConfigOverrides{}
	config, err := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(rules, overrides).ClientConfig()
	if err != nil {
		panic(err)
	}

	if config == nil {
		config = &rest.Config{}
	}

	client, err := dynamic.NewForConfig(config)
	if err != nil {
		panic(err)
	}

	return &Helper{
		client: client,
	}
}

// CreateNamespace creates a new namespace
func (k *Helper) CreateNamespace(t *testing.T, name string) {
	t.Helper()

	namespace := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "v1",
			"kind":       "Namespace",
			"metadata": map[string]interface{}{
				"name": name,
			},
		},
	}
	gvr := createGroupVersionResource("v1", "namespaces")
	_, err := k.client.Resource(gvr).Create(context.TODO(), namespace, metav1.CreateOptions{})
	if err != nil {
		t.Fatalf("Failed to create namespace %q: %v", name, err)
	}
}

// DeleteNamespace deletes a namespace
func (k *Helper) DeleteNamespace(t *testing.T, name string) {
	t.Helper()

	gvr := createGroupVersionResource("v1", "namespaces")
	err := k.client.Resource(gvr).Delete(context.TODO(), name, metav1.DeleteOptions{})
	if err != nil {
		t.Fatalf("Failed to delete namespace %q: %v", name, err)
	}
}

func createGroupVersionResource(gv, resource string) schema.GroupVersionResource {
	gvr, _ := schema.ParseGroupVersion(gv)
	return gvr.WithResource(resource)
}

// AssertNamespacedResourceExists will fail the current test if the resource doesn't exist in the
// specified namespace
func (k *Helper) AssertNamespacedResourceExists(t *testing.T, gv, resource, namespace, name string) {
	t.Helper()

	gvr := createGroupVersionResource(gv, resource)
	_, err := k.client.Resource(gvr).Namespace(namespace).Get(context.TODO(), name, metav1.GetOptions{})
	if errors.IsNotFound(err) {
		t.Errorf("Resource %s/%s does not exist", namespace, name)
		return
	}

	if err != nil {
		t.Errorf("Error when trying to get resource %s/%s: %v", namespace, name, err)
	}
}

// AssertResourceExists will fail the current test if the resource doesn't exist
func (k *Helper) AssertResourceExists(t *testing.T, gv, resource, name string) {
	t.Helper()

	gvr := createGroupVersionResource(gv, resource)
	_, err := k.client.Resource(gvr).Get(context.TODO(), name, metav1.GetOptions{})
	if errors.IsNotFound(err) {
		t.Errorf("Resource %s does not exist", name)
		return
	}

	if err != nil {
		t.Errorf("Error when trying to get resource %s: %v", name, err)
	}
}

// AssertNamespacedResourceDoesNotExist fails the test if the resource still exists in the namespace specified
func (k *Helper) AssertNamespacedResourceDoesNotExist(t *testing.T, gv, resource, namespace, name string) {
	t.Helper()

	gvr := createGroupVersionResource(gv, resource)

	for i := 1; i <= 3; i++ {
		_, err := k.client.Resource(gvr).Namespace(namespace).Get(context.TODO(), name, metav1.GetOptions{})
		if errors.IsNotFound(err) {
			return
		}

		if err != nil {
			t.Errorf("Error when trying to get resource %s/%s: %v", namespace, name, err)
			return
		}

		// NOTE some resources take a few seconds to delete so here we wait and retry so
		// we don't polute the tests with retry logic
		time.Sleep(time.Duration(math.Exp2(float64(i))) * time.Second)
	}

	t.Errorf("Resource %s/%s still exists", namespace, name)
}

// AssertResourceDoesNotExist fails the test if the resource still exists
func (k *Helper) AssertResourceDoesNotExist(t *testing.T, gv, resource, name string) {
	t.Helper()

	gvr := createGroupVersionResource(gv, resource)

	for i := 1; i <= 3; i++ {
		_, err := k.client.Resource(gvr).Get(context.TODO(), name, metav1.GetOptions{})
		if errors.IsNotFound(err) {
			return
		}

		if err != nil {
			t.Errorf("Error when trying to get resource %s: %v", name, err)
			return
		}

		// NOTE some resources take a second to delete so here we wait and retry so
		// we don't polute the tests with retry logic
		time.Sleep(time.Duration(math.Exp2(float64(i))) * time.Second)
	}

	t.Errorf("Resource %s still exists", name)
}
