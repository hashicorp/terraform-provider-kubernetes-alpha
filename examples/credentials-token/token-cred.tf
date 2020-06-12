# Example demonstrates authentication to the Kubernetes API via a service account token.
#
# After creating a service account, the token is available in a Secret resource associated to the service account.
# The secret also includes the cluster certificate authority required to securely access the API.
# Retrieve the token and CA from the Secret and paste them into the attributes below.
#
provider "kubernetes-alpha" {
  host = "https://192.168.64.51:8443"
  cluster_ca_certificate = file("ca.crt")
  token = "<token-string-goes-here>"
}

# Example resource
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