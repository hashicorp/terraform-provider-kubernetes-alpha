resource "kubedynamic_hcl_manifest" "test-ns" {
  manifest = {
    "apiVersion" = "v1"
    "kind" = "Namespace"
    "metadata" = {
        "name" = "test-ns"
    }
  }
}

resource "kubedynamic_hcl_manifest" "test-pod" {
  manifest = {
    "apiVersion" = "v1"
    "kind" = "Pod"
    "metadata" = {
      "labels" = {
        "app" = "nginx"
        "environment" = "production"
      }
      "name" = "label-demo"
      "namespace" = kubedynamic_hcl_manifest.test-ns.object.metadata.name
    }
    "spec" = {
      "containers" = [
        {
          "image" = "nginx:1.7.9"
          "name" = "nginx"
          "ports" = [
            {
              "containerPort" = 80
            },
          ]
        },
      ]
    }
  }
}
