variable "server_side_planning" {
  type = bool
  default = false
}

provider "kubernetes-alpha" {
  server_side_planning = var.server_side_planning

  config_path = "~/.kube/config"
}

resource "kubernetes_manifest" "test-namespace" {
  provider = kubernetes-alpha

  manifest = {
    "apiVersion" = "v1"
    "kind"       = "Namespace"
    "metadata" = {
      "name" = "tf-demo"
    }
  }
}