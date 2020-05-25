resource "kubernetes_service_account" "tfc-service-account" {
  metadata {
    name = "${var.namespace}-sync-workspace"
    namespace = var.namespace
    labels = {
      app = var.namespace
    }
  }
}