package provider

import (
	"testing"

	tfstatehelper "github.com/hashicorp/terraform-provider-kubernetes-alpha/test/helper/state"
)

func TestKubernetesManifest_WaitForFields(t *testing.T) {
	name := randName()
	namespace := randName()

	tf := tfhelper.RequireNewWorkingDir(t)
	defer func() {
		tf.RequireDestroy(t)
		tf.Close()
		k8shelper.AssertNamespacedResourceDoesNotExist(t, "v1", "pods", namespace, name)
	}()

	k8shelper.CreateNamespace(t, namespace)
	defer k8shelper.DeleteNamespace(t, namespace)

	tfvars := TFVARS{
		"server_side_planning": useServerSidePlanning,
		"namespace":            namespace,
		"name":                 name,
	}
	tfconfig := loadTerraformConfig(t, "wait_for_fields.tf", tfvars)
	tf.RequireSetConfig(t, tfconfig)
	tf.RequireInit(t)

	// TODO time this
	tf.RequireApply(t)

	k8shelper.AssertNamespacedResourceExists(t, "v1", "pods", namespace, name)

	tfstate := tfstatehelper.NewHelper(tf.RequireState(t))
	tfstate.AssertAttributeValues(t, tfstatehelper.AttributeValues{
		"kubernetes_manifest.test.wait_for": map[string]interface{}{
			"fields": map[string]interface{}{
				"status.containerStatuses.0.ready": "true",
			},
		},
	})
}
