package provider

import (
	"testing"

	tfstatehelper "github.com/hashicorp/terraform-provider-kubernetes-alpha/test/helper/state"
)

func TestKubernetesManifest_Deployment(t *testing.T) {
	name := randName()
	namespace := randName()

	tf := tfhelper.RequireNewWorkingDir(t)
	defer func() {
		tf.RequireDestroy(t)
		tf.Close()
		k8shelper.AssertNamespacedResourceDoesNotExist(t, "apps/v1", "deployments", namespace, name)
	}()

	k8shelper.CreateNamespace(t, namespace)
	defer k8shelper.DeleteNamespace(t, namespace)

	tfvars := TFVARS{
		"server_side_planning": useServerSidePlanning,
		"namespace":            namespace,
		"name":                 name,
	}
	tfconfig := loadTerraformConfig(t, "deployment.tf", tfvars)
	tf.RequireSetConfig(t, tfconfig)
	tf.RequireInit(t)
	tf.RequireApply(t)

	k8shelper.AssertNamespacedResourceExists(t, "apps/v1", "deployments", namespace, name)

	tfstate := tfstatehelper.NewHelper(tf.RequireState(t))
	tfstate.AssertAttributeValues(t, tfstatehelper.AttributeValues{
		"kubernetes_manifest.test.object.metadata.namespace":                                    namespace,
		"kubernetes_manifest.test.object.metadata.name":                                         name,
		"kubernetes_manifest.test.object.spec.template.spec.containers.0.name":                  "nginx",
		"kubernetes_manifest.test.object.spec.template.spec.containers.0.image":                 "nginx:1",
		"kubernetes_manifest.test.object.spec.template.spec.containers.0.ports.0.containerPort": 80,
	})
}
