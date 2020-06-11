provider "kubernetes-alpha" {
    config_path = "~/.kube/config"
}

resource "kubernetes_manifest" "test-configmap" {
  provider = kubernetes-alpha
  manifest = {
    "apiVersion" = "v1"
    "kind" = "ConfigMap"
    "metadata" = {
      "name" = "test-config"
      "namespace" = "default"
    }
    "data" = {
      "foo" = "bar"
    }
  }
}
