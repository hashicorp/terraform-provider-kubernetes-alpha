# Example demonstrates how to authenticate to a cluster API using client certificates
#
provider "kubernetes-alpha" {
  host = "https://192.168.246.148:8443"

  cluster_ca_certificate = file("~/.minikube/ca.crt")

  client_certificate = file("~/.minikube/profiles/minikube/client.crt")
  client_key = file("~/.minikube/profiles/minikube/client.key")
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