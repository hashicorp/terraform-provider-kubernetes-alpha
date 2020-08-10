provider "kubernetes-alpha" {
  config_path = "~/.kube/config"
  server_side_planning = true
}

resource "kubernetes_manifest" "example" {
  provider = kubernetes-alpha

  manifest = {
    apiVersion = "v1"
    kind       = "Pod"

    metadata = {
      name      = "example-pod"
      namespace = "default"

      labels = {
        app = "nginx"
      }
    }

    spec = {
      containers = [
        {
          image = "nginx:1.19"
          name  = "nginx"
        },
      ]
    }
  }

  wait_for = {
    fields = {
      "status.phase" = "Running"
    }
  }
}
