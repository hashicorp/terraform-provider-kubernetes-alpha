# terraform-k8s-module

A Terraform module for managing the HashiCorp Terraform-k8s operator with the Kubernetes and Kubernetes-alpha providers. 

## Usage
```
provider "kubernetes" {}
provider "kubernetes-alpha" {}

resource "kubernetes_namespace" "example" {
  metadata {
    name = var.namespace
  }
}

module "terraform-k8s" {
  source = "github.com/hashicorp/terraform-provider-kubernetes-alpha/tree/dev/examples/terraform-k8s"

namespace       = var.namespace
tfc_credentials = file(var.tfc_credentials)
access_key_id     = var.access_key_id
secret_acess_key = var.secret_acess_key
}
```