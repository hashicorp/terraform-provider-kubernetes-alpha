# Terraform provider for Kubernetes (ALPHA)
![Terraform](https://img.shields.io/badge/-Terraform-623CE4?logo=data:image/svg+xml;base64,PHN2ZyB3aWR0aD0iMTIiIGhlaWdodD0iMTIiIHZpZXdCb3g9IjAgMCAxMiAxMiIgZmlsbD0ibm9uZSIgeG1sbnM9Imh0dHA6Ly93d3cudzMub3JnLzIwMDAvc3ZnIj4KPHBhdGggZmlsbC1ydWxlPSJldmVub2RkIiBjbGlwLXJ1bGU9ImV2ZW5vZGQiIGQ9Ik00LjU0NDc0IDIuMDEzNzZMNy4zOTk2NyAzLjY2MTczVjYuOTU3NjdMNC41NDQ3NCA1LjMwOTdWMi4wMTM3NloiIGZpbGw9IndoaXRlIi8+CjxwYXRoIGZpbGwtcnVsZT0iZXZlbm9kZCIgY2xpcC1ydWxlPSJldmVub2RkIiBkPSJNNy43MTE5OCAzLjY2MTcyVjYuOTU3NjZMMTAuNTY2OSA1LjMwOTY5VjIuMDEyM0w3LjcxMTk4IDMuNjYxNzJaIiBmaWxsPSIjRjJGMkYyIi8+CjxwYXRoIGZpbGwtcnVsZT0iZXZlbm9kZCIgY2xpcC1ydWxlPSJldmVub2RkIiBkPSJNMS4zNzY0IDAuMTcyODUyVjMuNDY4NzlMNC4yMzEzNCA1LjExODIxVjEuODIwODJMMS4zNzY0IDAuMTcyODUyWiIgZmlsbD0id2hpdGUiLz4KPHBhdGggZmlsbC1ydWxlPSJldmVub2RkIiBjbGlwLXJ1bGU9ImV2ZW5vZGQiIGQ9Ik00LjU0NDc0IDguOTY2ODFMNy4zOTgyMiAxMC42MTYyVjcuMzQwNlY3LjMxODg0TDQuNTQ0NzQgNS42NzA4N1Y4Ljk2NjgxWiIgZmlsbD0id2hpdGUiLz4KPC9zdmc+Cg==) ![Status: Experimental](https://img.shields.io/badge/status-experimental-EAAA32) [![Releases](https://img.shields.io/github/release/hashicorp/terraform-provider-kubernetes-alpha/all.svg?style=flat-square)](https://github.com/hashicorp/terraform-provider-kubernetes-alpha/releases)
[![LICENSE](https://img.shields.io/github/license/hashicorp/terraform-provider-kubernetes-alpha.svg?style=flat-square)](https://github.com/hashicorp/terraform-provider-kubernetes-alpha/blob/master/LICENSE)
![unit tests](https://github.com/hashicorp/terraform-provider-kubernetes-alpha/workflows/unit%20tests/badge.svg)
![acceptance tests](https://github.com/hashicorp/terraform-provider-kubernetes-alpha/workflows/acceptance%20tests/badge.svg)

A Terraform provider for Kubernetes which supports all API resources in a generic fashion.

This provider allows you to describe any Kubernetes resource using HCL. See [Moving from YAML to HCL](#moving-from-yaml-to-hcl) if you have YAML you want to use with the provider.

Please regard this project as experimental. It still requires extensive testing and polishing to mature into production-ready quality. Please file issues generously and detail your experience while using the provider. We encourage all types of feedback.

## Requirements

* [Terraform](https://www.terraform.io/downloads.html) version 0.12.x +
* [Kubernetes](https://kubernetes.io/docs/reference) version 1.17.x +
* [Go](https://golang.org/doc/install) version 1.14.x

## Usage Example

* Create a Kubernetes ConfigMap
```hcl
provider "kubernetes-alpha" {}

resource "kubernetes_manifest" "test-configmap" {
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

resource "kubernetes_manifest" "test-crd" {
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

## Moving from YAML to HCL

The `manifest` attribute of the `kubernetes_manifest` resource accepts any arbitrary Kubernetes API object, using Terraform's [map](https://www.terraform.io/docs/configuration/expressions.html#map) syntax. If you have YAML you want to use with this provider, we recommend that you convert it to a map as an initial step, rather than using `yamldecode()` inside the resource block. 

You can quickly convert a single YAML file to an HCL map using this one liner:

```
echo 'yamldecode(file("test.yaml"))' | terraform console
```

There is also an experimental command line tool [tfk8s](https://github.com/jrhouston/tfk8s) which you can use to convert Kubernetes YAML manifests to complete Terraform configurations.

## Building and installing

As we do not yet publish releases for this provider to the registry, you have to install it manualy.

Make sure you have a supported version of Go installed and working.

Checkout or download this repository, then open a terminal and change to its directory.

### Using `GOBIN` and `terraform.d/plugins`
```
make install
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

### Using `-plugin-dir` 

Alternative, you can run:

```
make build
```

This will place the provider binary in the top level of the provider directory. You can then use it with terraform by specifying the `-plugin-dir` option when running `terraform init`

```
terraform init -plugin-dir /path/to/terraform-provider-alpha
```

## Experimental Status

By using the software in this repository (the "Software"), you acknowledge that: (1) the Software is still in development, may change, and has not been released as a commercial product by HashiCorp and is not currently supported in any way by HashiCorp; (2) the Software is provided on an "as-is" basis, and may include bugs, errors, or other issues;  (3) the Software is NOT INTENDED FOR PRODUCTION USE, use of the Software may result in unexpected results, loss of data, or other unexpected results, and HashiCorp disclaims any and all liability resulting from use of the Software; and (4) HashiCorp reserves all rights to make all decisions about the features, functionality and commercial release (or non-release) of the Software, at any time and without any obligation or liability whatsoever.
