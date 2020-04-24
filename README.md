# Generic Terraform provider for Kubernetes (ALPHA)

A Terraform provider for Kubernetes which supports all API resources in a generic fashion.

Resources for this provider are described in HCL. It also experimentally supports native Kubernetes YAML manifests with limited functionality.

## Experimental status (please read)

This new provider represents a significant departure from the established practice of developing Terraform providers. It does not make use of the [provider SDK](https://github.com/hashicorp/terraform-plugin-sdk) as all other providers currently do. This shift was necesary in order to leverege certain low-level features introduced in Terraform with version 0.12 which are currently not reflected in the SDK. Such features include a rich type system which allows for dynamically typed resource attributes. It was needed in order to implement variable-schema Kubrenetes resources like Custom Resource Definitions.

As a consequence of not using the provider SDK, certain "typical" features of Terraform such as planning changes have had to be partially reimplemented in the provider. The state of this implemtations is still evolving and as such may not yeld as smooth of an experince as other more mature providers. Particularly resource updating has rough edges that are being actively worked on.

Please regard this project as experimental. It still requires extensive testing and polishing to mature into production-ready quality.

DO NOT USE IN PRODCUTION!

Please file issues generously and detail your experience while using the provider. We encourage all types of feedback.

## Requirements

* [Terraform](https://www.terraform.io/downloads.html) version 0.12.x +
* [Kubernetes](https://kubernetes.io/docs/reference) version 1.17.x +
* [Go](https://golang.org/doc/install) version 1.14.x

## Usage Example

* Create a Kubernetes ConfigMap
```hcl
provider "kubernetes-alpha" {}

resource "kubernetes_manifest_hcl" "test-configmap" {
  provider = kubernetes-alpha

  manifest = {
    "apiVersion" = "v1"
    "kind" = "ConfigMap"
    "metadata" = {
      "name" = "test-config"
      "namespace" = "default"
    }
    "data" = {
      "foo" = "bar"
    }
  }
}
```

* Create a Kubernetes Custom Resource Definition

```hcl
provider "kubernetes-alpha" {}

resource "kubernetes_manifest_hcl" "test-crd" {
  provider = kubernetes-alpha

  manifest = {
    apiVersion = "apiextensions.k8s.io/v1"
    kind = "CustomResourceDefinition"
    metadata = {
      name = "testcrds.hashicorp.com"
    }
    spec = {
      group = "hashicorp.com"
      names = {
        kind = "TestCrd"
        plural = "testcrds"
      }
      scope = "Namespaced"
      versions = [{
        name = "v1"
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
```

## Building and installing

As we do not yet publish releases for this provider to the registry, you have to install it manualy.

Make sure you have a supported version of Go installed and working.

Checkout or download this repository, then open a terminal and change to its directory.

Build just as any other Go project:
```
~/terraform-provider-kubernetes-alpha » go install
```
This will place the provider binary in the GOBIN directory. You can determine this location using the `go env GOBIN` command.

You then need to link this binary file into a filesystem location where Terraform can find it. One such location is `$HOME/.terraform.d/plugins/`. More on this topic [here](https://www.terraform.io/docs/extend/how-terraform-works.html#discovery)

Create the link with following commands:
```
mkdir -p $HOME/.terraform.d/plugins
ln -s $(go env GOBIN)/terraform-provider-kubernetes-alpha $HOME/.terraform.d/plugins/terraform-provider-kubernetes-alpha
```

You are now ready to use the provider. You can find example TF configurations in this repository under the `./examples`.

Don't forget to run `terraform init` in the TF configuration directory to allow Terraform to detect the provider binary.
