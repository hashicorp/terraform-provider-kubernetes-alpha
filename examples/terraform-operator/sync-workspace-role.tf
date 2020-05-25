resource "kubernetes_role" "tfc-role" {
  metadata {
    name = "${var.namespace}-sync-workspace"
    namespace = var.namespace
    labels = {
      app = var.namespace
    }
  }

  rule {
    api_groups = [""]
    resources  = ["pods", "services", "services/finalizers", "endpoints", "persistentvolumeclaims", "events", "configmaps", "secrets"]
    verbs      = ["*"]
  }
  rule {
    api_groups = ["apps"]
    resources  = ["deployments", "daemonsets", "replicasets", "statefulsets"]
    verbs      = ["*"]
  }
  rule {
    api_groups = ["monitoring.coreos.com"]
    resources  = ["servicemonitors"]
    verbs      = ["get", "create"]
  }

  rule {
    api_groups     = ["apps"]
    resource_names = ["terraform-k8s"]
    resources      = ["deployments/finalizers"]
    verbs          = ["update"]
  }

  rule {
    api_groups = [""]
    resources  = ["pods"]
    verbs      = ["get"]
  }

  rule {
    api_groups = ["apps"]
    resources  = ["replicasets"]
    verbs      = ["get"]
  }

  rule {
    api_groups = ["app.terraform.io"]
    resources  = ["*", "workspaces"]
    verbs      = ["*"]
  }
}