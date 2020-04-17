provider "kubernetes-alpha" {
}

resource "kubernetes_manifest_hcl" "test-namespace" {
  provider = kubernetes-alpha

  manifest = {
    "apiVersion" = "v1"
    "kind" = "Namespace"
    "metadata" = {
      "name" = "tf-demo"
    }
  }
}