variable "server_side_planning" {
  type = bool
  default = false
}

provider "kubernetes-alpha" {
  server_side_planning = var.server_side_planning

  config_path = "~/.kube/config"
}

resource "kubernetes_manifest" "test-configmap" {
  provider = kubernetes-alpha
  manifest = {
    "apiVersion" = "v1"
    "kind"       = "ConfigMap"
    "metadata" = {
      "name"      = "test-config"
      "namespace" = "default"
      "labels" = {
        "app"         = "test-app"
        "environment" = "production"
      }
    }
    "data" = {
      "foo" = "bar"
    }
  }
}
