variable "kubeconfig" {}

provider "kubedynamic" {
    config_file = var.kubeconfig
}

resource "kubedynamic_yaml_manifest" "test-crd" {
  manifest = file("${path.module}/test-crd.yaml")
}
