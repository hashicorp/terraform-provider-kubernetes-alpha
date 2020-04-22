variable "kubeconfig" {}

provider "kubernetes-alpha" {
  config_file          = var.kubeconfig
  server_side_planning = true
}

# PodSecurityPolicy only works on Kubernetes 1.17+
resource "kubernetes_manifest_hcl" "psp" {
  provider = kubernetes-alpha
  manifest = {
    "apiVersion" = "policy/v1beta1"
    "kind"       = "PodSecurityPolicy"
    "metadata" = {
      "name"      = "example"
      "namespace" = "default"
    }
    "spec" = {
      "fsGroup" = {
        "rule" = "RunAsAny"
      }
      "privileged" = false
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