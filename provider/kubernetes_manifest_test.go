package provider

import (
	"fmt"
	"testing"
)

func TestAccKubernetesManifestBasicConfigMap(t *testing.T) {
	wd := helper.RequireNewWorkingDir(t)

	namespace := randName()
	createKubernetesNamespace(t, namespace)

	name := randName()
	tfconfig := testAccManifestBasicConfigMap(namespace, name)
	wd.RequireSetConfig(t, tfconfig)

	wd.RequireInit(t)
	wd.RequireApply(t)

	assertKubernetesNamespacedResourceExists(t, "v1", "configmaps", namespace, name)

	state := wd.RequireState(t)
	object := getObjectFromResourceState(t, state, "kubernetes_manifest.test")
	assertObjectFieldEqual(t, object, "metadata.namespace", namespace)
	assertObjectFieldEqual(t, object, "metadata.name", name)
	assertObjectFieldEqual(t, object, "data.foo", "bar")

	wd.Destroy()
	assertKubernetesNamespacedResourceNotExists(t, "v1", "configmaps", namespace, name)
	deleteKubernetesNamespace(t, namespace)

	wd.Close()
}

func testAccManifestBasicConfigMap(namespace, name string) string {
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
