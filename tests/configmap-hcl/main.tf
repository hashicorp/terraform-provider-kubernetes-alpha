variable "kubeconfig" {}

provider "kubedynamic" {
  config_file = var.kubeconfig
}

# resource "kubedynamic_hcl_manifest" "test-namespace" {
#   manifest = {
#     "apiVersion" = "v1"
#     "kind" = "Namespace"
#     "metadata" = {
#       name = "test-ns"
#     }
#   }
# }

resource "kubedynamic_hcl_manifest" "test-crd" {
  manifest = {
    "apiVersion" = "v1"
    "kind" = "ConfigMap"
    "metadata" = {
      "name" = "test-config"
      "namespace" = "test-ns"
    }
    "data" = {
      "foo" = "bar"
    }
  }
}
