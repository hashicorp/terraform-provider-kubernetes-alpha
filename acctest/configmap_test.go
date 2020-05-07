package provider

import (
	"fmt"
	"testing"

	statehelper "github.com/hashicorp/terraform-provider-kubernetes-alpha/acctest/helper/state"
)

// This test case tests a ConfigMap but also is a demonstration of some the assert functions
// available in the test helper
func TestKubernetesManifest_ConfigMap(t *testing.T) {
	name := randName()
	namespace := randName()

	tf := binhelper.RequireNewWorkingDir(t)
	defer func() {
		tf.RequireDestroy(t)
		tf.Close()
		kubernetesHelper.AssertNamespacedResourceDoesNotExist(t, "v1", "configmaps", namespace, name)
	}()

	kubernetesHelper.CreateNamespace(t, namespace)
	defer kubernetesHelper.DeleteNamespace(t, namespace)

	tfconfig := testKubernetesManifestConfig_ConfigMap(namespace, name)
	tf.RequireSetConfig(t, tfconfig)
	tf.RequireInit(t)
	tf.RequireApply(t)

	kubernetesHelper.AssertNamespacedResourceExists(t, "v1", "configmaps", namespace, name)

	tfstate := statehelper.Wrap(t, tf.RequireState(t))
	tfstate.AssertAttributeEqual("kubernetes_manifest.test.object.metadata.namespace", namespace)
	tfstate.AssertAttributeEqual("kubernetes_manifest.test.object.metadata.name", name)
	tfstate.AssertAttributeEqual("kubernetes_manifest.test.object.data.foo", "bar")

	tfconfigModified := testKubernetesManifestConfig_ConfigMapModified(namespace, name)
	tf.RequireSetConfig(t, tfconfigModified)
	tf.RequireApply(t)

	tfstate = statehelper.Wrap(t, tf.RequireState(t))
	tfstate.AssertAttributeEqual("kubernetes_manifest.test.object.metadata.namespace", namespace)
	tfstate.AssertAttributeEqual("kubernetes_manifest.test.object.metadata.name", name)
	tfstate.AssertAttributeEqual("kubernetes_manifest.test.object.metadata.annotations.test", "1")
	tfstate.AssertAttributeEqual("kubernetes_manifest.test.object.metadata.labels.test", "2")
	tfstate.AssertAttributeEqual("kubernetes_manifest.test.object.data.foo", "bar")
	tfstate.AssertAttributeEqual("kubernetes_manifest.test.object.data.fizz", "buzz")

	tfstate.AssertAttributeLen("kubernetes_manifest.test.object.metadata.labels", 1)
	tfstate.AssertAttributeLen("kubernetes_manifest.test.object.metadata.annotations", 1)

	tfstate.AssertAttributeNotEmpty("kubernetes_manifest.test.object.metadata.labels.test")

	tfstate.AssertAttributeDoesNotExist("kubernetes_manifest.test.spec")
}

func testKubernetesManifestConfig_ConfigMap(namespace, name string) string {
	return fmt.Sprintf(`	
resource "kubernetes_manifest" "test" {
  provider = kubernetes-alpha

  manifest = {
    "apiVersion" = "v1"
    "kind" = "ConfigMap"
    "metadata" = {
	  "name" = %q
	  "namespace" = %q	
    }
    "data" = {
      "foo" = "bar"
    }
  }
}`, name, namespace)
}

func testKubernetesManifestConfig_ConfigMapModified(namespace, name string) string {
	return fmt.Sprintf(`	
resource "kubernetes_manifest" "test" {
  provider = kubernetes-alpha

  manifest = {
    "apiVersion" = "v1"
    "kind" = "ConfigMap"
    "metadata" = {
	  "name" = %q
	  "namespace" = %q
	  "annotations" = {
		"test" = "1"
	  }
	  "labels" = {
	    "test" = "2"
	  }
    }
    "data" = {
	  "foo" = "bar"
	  "fizz" = "buzz"
    }
  }
}`, name, namespace)
}
