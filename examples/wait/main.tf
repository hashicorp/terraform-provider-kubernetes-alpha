provider "kubernetes-alpha" {
  config_path = "~/.kube/config"
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
          name  = "nginx"
          image = "nginx:1.19"

          readinessProbe = {
            initialDelaySeconds = 10

            httpGet = {
              path = "/"
              port = 80
            }
          }
        }
      ]
    }
  }

  wait_for = {
    fields = {
      "status.containerStatuses.0.ready"        = "true",
      "status.containerStatuses.0.restartCount" = "0",
      "status.podIP"                            = "^(\\d+(\\.|$)){4}",
      "status.phase"                            = "Running",
    }
  }
}
