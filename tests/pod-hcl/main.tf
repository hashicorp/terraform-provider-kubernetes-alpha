provider "kubernetes-alpha" {
}

resource "kubernetes_hcl_manifest" "test-pod" {
  provider = kubernetes-alpha
  manifest = {
    "apiVersion" = "v1"
    "kind" = "Pod"
    "metadata" = {
      "name" = "label-demo"
      "namespace" = "default"
      "labels" = {
        "app" = "nginx"
        "environment" = "production"
      }
    }
    "spec" = {
      "containers" = [
        {
          "image" = "nginx:1.7.9"
          "name" = "nginx"
          "ports" = [
            {
              "containerPort" = 80
              "protocol" = "TCP"
            },
          ]
        },
      ]
    }
  }
}
