package provider

import (
	"context"
	"fmt"
	"math/rand"
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"

	tfjson "github.com/hashicorp/terraform-json"
	tftest "github.com/hashicorp/terraform-plugin-test"

	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	_ "k8s.io/client-go/plugin/pkg/client/auth"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

var providerName = "kubernetes-alpha"
var helper *tftest.Helper
var kubernetesClient dynamic.Interface

func TestMain(m *testing.M) {
	if tftest.RunningAsPlugin() {
		Serve()
		os.Exit(0)
		return
	}

	sourceDir, err := os.Getwd()
	if err != nil {
		panic(err)
	}

	client, err := configureKubernetesClient()
	if err != nil {
		panic(err)
	}
	kubernetesClient = client

	helper = tftest.AutoInitProviderHelper(providerName, sourceDir)
	defer helper.Close()

	exitcode := m.Run()
	os.Exit(exitcode)
}

func configureKubernetesClient() (dynamic.Interface, error) {
	rules := clientcmd.NewDefaultClientConfigLoadingRules()
	overrides := &clientcmd.ConfigOverrides{}
	config, err := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(rules, overrides).ClientConfig()
	if err != nil {
		return nil, err
	}

	if config == nil {
		config = &rest.Config{}
	}

	client, err := dynamic.NewForConfig(config)
	if err != nil {
		return nil, err
	}

	return client, nil
}

func createGroupVersionResource(gv, resource string) schema.GroupVersionResource {
	var group, groupVersion string
	gvparts := strings.Split(gv, "/")
	if len(gvparts) < 2 {
		group = ""
		groupVersion = gv
	} else {
		group = gvparts[0]
		groupVersion = gvparts[1]
	}

	return schema.GroupVersionResource{
		Group:    group,
		Version:  groupVersion,
		Resource: resource,
	}
}

func assertKubernetesNamespacedResourceExists(t *testing.T, gv, resource, namespace, name string) {
	gvr := createGroupVersionResource(gv, resource)
	_, err := kubernetesClient.Resource(gvr).Namespace(namespace).Get(context.TODO(), name, metav1.GetOptions{})
	if errors.IsNotFound(err) {
		t.Fatalf("Resource %s/%s does not exist", namespace, name)
		return
	}

	if err != nil {
		t.Fatalf("Error when trying to get resource %s/%s: %v", namespace, name, err)
	}
}

func assertKubernetesNamespacedResourceNotExists(t *testing.T, gv, resource, namespace, name string) {
	gvr := createGroupVersionResource(gv, resource)
	_, err := kubernetesClient.Resource(gvr).Namespace(namespace).Get(context.TODO(), name, metav1.GetOptions{})
	if errors.IsNotFound(err) {
		return
	}

	if err != nil {
		t.Fatalf("Error when trying to get resource %s/%s: %v", namespace, name, err)
		return
	}

	t.Fatalf("Resource %s/%s still exists", namespace, name)
}

func createKubernetesNamespace(t *testing.T, name string) {
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
	_, err := kubernetesClient.Resource(gvr).Create(context.TODO(), namespace, metav1.CreateOptions{})
	if err != nil {
		t.Fatalf("Failed to create namespace %q: %v", name, err)
	}
}

func deleteKubernetesNamespace(t *testing.T, name string) {
	gvr := createGroupVersionResource("v1", "namespaces")
	err := kubernetesClient.Resource(gvr).Delete(context.TODO(), name, metav1.DeleteOptions{})
	if err != nil {
		t.Fatalf("Failed to delete namespace %q: %v", name, err)
	}
}

var letters = []rune("abcdefghijklmnopqrstuvwxyz")

func randName() string {
	b := make([]rune, 10)
	for i := range b {
		b[i] = letters[rand.Intn(len(letters))]
	}
	return fmt.Sprintf("tf-acc-test-%s", string(b))
}

// getObjectFromResourceState will pull out the value of the `object` attribute of the resource and return it
// as a map[string]interface{}.
func getObjectFromResourceState(t *testing.T, state *tfjson.State, resourceAddr string) map[string]interface{} {
	for _, r := range state.Values.RootModule.Resources {
		if r.Address == resourceAddr {
			value, ok := r.AttributeValues["object"].(map[string]interface{})
			if !ok {
				t.Fatalf("Could not find get `object` attribute from %q", resourceAddr)
			}
			return value
		}
	}

	t.Fatalf("Could not find resource %q in state", resourceAddr)
	return nil
}

// findFieldValue will return the value of a field in the object using dot notation
func findFieldValue(object map[string]interface{}, fieldPath string) (interface{}, error) {
	pathKeys := strings.Split(fieldPath, ".")
	fieldKey := pathKeys[len(pathKeys)-1]
	parentFieldPathKeys := pathKeys[:len(pathKeys)-1]

	m := object
	for _, key := range parentFieldPathKeys {
		// FIXME need to handle arrays like: spec.containers.0.name
		v, ok := m[key].(map[string]interface{})
		if !ok {
			return nil, fmt.Errorf("Field %q does not exist", fieldPath)
		}
		m = v
	}

	value, ok := m[fieldKey].(interface{})
	if !ok {
		return nil, fmt.Errorf("Field %q does not exist", fieldPath)
	}

	return value, nil
}

func assertObjectFieldEqual(t *testing.T, object map[string]interface{}, fieldPath string, expectedValue interface{}) {
	actualValue, err := findFieldValue(object, fieldPath)
	if err != nil {
		t.Fatalf(err.Error())
	}

	assert.Equal(t, expectedValue, actualValue)
}
