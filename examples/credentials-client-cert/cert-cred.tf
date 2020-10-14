# Example demonstrates how to authenticate to a cluster API using client certificates
#
variable "minikube_ip" {
  type = string
}

variable "server_side_planning" {
  type = bool
  default = false
}

provider "kubernetes-alpha" {
  server_side_planning = var.server_side_planning

  host = "https://${var.minikube_ip}:8443"

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