package provider

import (
	"fmt"
	"strings"
	"testing"

	tfstatehelper "github.com/hashicorp/terraform-provider-kubernetes-alpha/test/helper/state"
)

func TestKubernetesManifest_CustomResourceDefinition(t *testing.T) {
	kind := strings.Title(randString(8))
	plural := strings.ToLower(kind) + "s"
	group := "terraform.io"
	version := "v1"
	name := fmt.Sprintf("%s.%s", plural, group)

	tf := tfhelper.RequireNewWorkingDir(t)
	defer func() {
		tf.RequireDestroy(t)
		tf.Close()
		k8shelper.AssertResourceDoesNotExist(t, "apiextensions.k8s.io/v1", "customresourcedefinitions", name)
	}()

	tfvars := TFVARS{
		"server_side_planning": useServerSidePlanning,
		"kind":                 kind,
		"plural":               plural,
		"group":                group,
		"group_version":        version,
	}
	tfconfig := loadTerraformConfig(t, "customresourcedefinition.tf", tfvars)
	tf.RequireSetConfig(t, tfconfig)
	tf.RequireInit(t)
	tf.RequireApply(t)

	k8shelper.AssertResourceExists(t, "apiextensions.k8s.io/v1", "customresourcedefinitions", name)

	tfstate := tfstatehelper.NewHelper(tf.RequireState(t))
	tfstate.AssertAttributeValues(t, tfstatehelper.AttributeValues{
		"kubernetes_manifest.test.object.metadata.name":           name,
		"kubernetes_manifest.test.object.spec.group":              group,
		"kubernetes_manifest.test.object.spec.names.kind":         kind,
		"kubernetes_manifest.test.object.spec.names.plural":       plural,
		"kubernetes_manifest.test.object.spec.scope":              "Namespaced",
		"kubernetes_manifest.test.object.spec.versions.0.name":    version,
		"kubernetes_manifest.test.object.spec.versions.0.served":  true,
		"kubernetes_manifest.test.object.spec.versions.0.storage": true,
		"kubernetes_manifest.test.object.spec.versions.0.schema": map[string]interface{}{
			"openAPIV3Schema": map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"data": map[string]interface{}{
						"type": "string",
					},
					"refs": map[string]interface{}{
						"type": "number",
					},
				},
			},
		},
	})
}
