variable "kubeconfig" {}

provider "kubedynamic" {
  config_file = var.kubeconfig
}

resource "kubedynamic_hcl_manifest" "test-crd" {
  manifest = {
    apiVersion = "apiextensions.k8s.io/v1"
    kind = "CustomResourceDefinition"
    objectMeta = {
      name = "testcrds.somesan.de"
    }
    spec = {
      group = "somesan.de"
      names = {
        kind = "TestCrd"
        plural = "testcrds"
      }
      scope = "Namespaced"
      versions = [
        {
          name = "v1alpha"
          served = true
          storage = true
        },
      ]
    }
  }
}
