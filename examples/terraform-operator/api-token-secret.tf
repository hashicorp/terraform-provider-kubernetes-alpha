resource "kubernetes_secret" "tfc-api-token" {
  metadata {
    name      = "terraformrc"
    namespace = var.namespace
    labels = {
      app = var.namespace
    }
  }

  data = {
    credentials = var.tfc_credentials
  }
}