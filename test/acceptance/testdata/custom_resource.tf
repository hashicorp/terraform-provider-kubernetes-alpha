provider "kubernetes-alpha" {
  server_side_planning = var.server_side_planning
}

resource "kubernetes_manifest" "test_crd" {
  provider = kubernetes-alpha

  manifest = {
    apiVersion = "apiextensions.k8s.io/v1"
    kind = "CustomResourceDefinition"
    metadata = {
      name = "${var.plural}.${var.group}"
    }
    spec = {
      group = var.group
      names = {
        kind = var.kind
        plural = var.plural
      }
      scope = "Namespaced"
      versions = [{
        name = var.group_version
        served = true
        storage = true
        schema = {
          openAPIV3Schema = {
            type = "object"
            properties = {
              data = {
                type = "string"
              }
              refs = {
                type = "number"
              }
            }
          }
        }
      }]
    }
  }
}

resource "kubernetes_manifest" "test" {
  provider = kubernetes-alpha

  depends_on = [
    kubernetes_manifest.test_crd
  ]

  manifest = {
    apiVersion = "${var.group}/${var.group_version}"
    kind = var.kind
    metadata = {
      namespace = var.namespace
      name = var.name
    }
    data = "this is a test"
  }
}

