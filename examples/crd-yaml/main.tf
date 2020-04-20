variable "kubeconfig" {}

provider "kubernetes-alpha" {
    config_file = var.kubeconfig
}

resource "kubernetes_yaml_manifest" "test-crd" {
  manifest = file("${path.module}/test-crd.yaml")
}
