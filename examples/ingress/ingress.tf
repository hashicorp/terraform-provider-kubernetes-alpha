variable "server_side_planning" {
  type = bool
  default = false
}

provider "kubernetes-alpha" {
  server_side_planning = var.server_side_planning
  
  config_path = "~/.kube/config"
}

resource "kubernetes_manifest" "test-ingress" {
  provider = kubernetes-alpha

  manifest = {
    "apiVersion" = "networking.k8s.io/v1beta1"
    "kind"       = "Ingress"
    "metadata" = {
      "annotations" = {
        "nginx.ingress.kubernetes.io/rewrite-target" = "/$1"
      }
      "name"      = "example-ingress"
      "namespace" = "default"
    }
    "spec" = {
      "rules" = [
        {
          "host" = "hello-world.info"
          "http" = {
            "paths" = [
              {
                "backend" = {
                  "serviceName" = "web"
                  "servicePort" = "http"
                }
                "path" = "/"
              },
            ]
          }
        },
      ]
    }
  }
}