variable "server_side_planning" {
  type = bool
  default = false
}

provider "kubernetes-alpha" {
  server_side_planning = var.server_side_planning

  config_path = "~/.kube/config"
}

# PodSecurityPolicy only works on Kubernetes 1.17+
resource "kubernetes_manifest" "psp" {
  provider = kubernetes-alpha
  manifest = {
    "apiVersion" = "policy/v1beta1"
    "kind"       = "PodSecurityPolicy"
    "metadata" = {
      "name"      = "example"
    }
    "spec" = {
      "fsGroup" = {
        "rule" = "RunAsAny"
      }
      "runAsUser" = {
        "rule" = "RunAsAny"
      }
      "seLinux" = {
        "rule" = "RunAsAny"
      }
      "supplementalGroups" = {
        "rule" = "RunAsAny"
      }
      "volumes" = [
        "*",
      ]
    }
  }
}
