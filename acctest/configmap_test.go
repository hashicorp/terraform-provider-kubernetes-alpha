package provider

import (
	"fmt"
	"testing"
)

func TestKubernetesManifest_ConfigMap(t *testing.T) {
	wd := helper.RequireNewWorkingDir(t)
	defer func() {
		wd.RequireDestroy(t)
		wd.Close()
	}()

	namespace := randName()
	createKubernetesNamespace(t, namespace)
	defer deleteKubernetesNamespace(t, namespace)

	name := randName()
	tfconfig := testKubernetesManifestConfig_ConfigMap(namespace, name)
	wd.RequireSetConfig(t, tfconfig)
	wd.RequireInit(t)
	wd.RequireApply(t)

	assertKubernetesNamespacedResourceExists(t, "v1", "configmaps", namespace, name)

	state := wd.RequireState(t)
	object := getObjectAttributeFromResourceState(t, state, "kubernetes_manifest.test")
	assertObjectFieldEqual(t, object, "metadata.namespace", namespace)
	assertObjectFieldEqual(t, object, "metadata.name", name)
	assertObjectFieldEqual(t, object, "data.foo", "bar")

	tfconfigModified := testKubernetesManifestConfig_ConfigMapModified(namespace, name)
	wd.RequireSetConfig(t, tfconfigModified)
	wd.RequireApply(t)

	state = wd.RequireState(t)
	object = getObjectAttributeFromResourceState(t, state, "kubernetes_manifest.test")
	assertObjectFieldEqual(t, object, "metadata.namespace", namespace)
	assertObjectFieldEqual(t, object, "metadata.name", name)
	assertObjectFieldEqual(t, object, "metadata.annotations.test", "1")
	assertObjectFieldEqual(t, object, "metadata.labels.test", "2")
	assertObjectFieldEqual(t, object, "data.foo", "bar")
	assertObjectFieldEqual(t, object, "data.fizz", "buzz")
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
