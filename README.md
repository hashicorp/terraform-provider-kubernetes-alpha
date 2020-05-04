# Terraform provider for Kubernetes (ALPHA)
![](https://img.shields.io/:Terraform-experiment-000000.svg?labelColor=623CE4&logo=data:image/svg%2bxml;base64,PD94bWwgdmVyc2lvbj0iMS4wIiBlbmNvZGluZz0idXRmLTgiPz4KPCEtLSBHZW5lcmF0b3I6IEFkb2JlIElsbHVzdHJhdG9yIDIxLjEuMCwgU1ZHIEV4cG9ydCBQbHVnLUluIC4gU1ZHIFZlcnNpb246IDYuMDAgQnVpbGQgMCkgIC0tPgo8c3ZnIHZlcnNpb249IjEuMSIgaWQ9IkxheWVyXzEiIHhtbG5zPSJodHRwOi8vd3d3LnczLm9yZy8yMDAwL3N2ZyIgeG1sbnM6eGxpbms9Imh0dHA6Ly93d3cudzMub3JnLzE5OTkveGxpbmsiIHg9IjBweCIgeT0iMHB4IgoJIHZpZXdCb3g9IjAgMCAxMDYuOSAxMTMuMSIgc3R5bGU9ImVuYWJsZS1iYWNrZ3JvdW5kOm5ldyAwIDAgMTA2LjkgMTEzLjE7IiB4bWw6c3BhY2U9InByZXNlcnZlIj4KPHN0eWxlIHR5cGU9InRleHQvY3NzIj4KCS5zdDB7ZmlsbDojRkZGRkZGO30KPC9zdHlsZT4KPHRpdGxlPmlBc3NldCAxPC90aXRsZT4KPGcgaWQ9IkxheWVyXzIiPgoJPGcgaWQ9IkxvZ28iPgoJCTxwb2x5Z29uIGNsYXNzPSJzdDAiIHBvaW50cz0iNDQuNSwwIDAsMjUuNyAwLDI1LjcgMCw4Ny40IDE2LjcsOTcuMSAxNi43LDM1LjMgNDQuNSwxOS4zIAkJIi8+CgkJPHBvbHlnb24gY2xhc3M9InN0MCIgcG9pbnRzPSI2Mi4zLDAgNjIuMyw0OS4yIDQ0LjUsNDkuMiA0NC41LDMwLjggMjcuOCw0MC41IDI3LjgsMTAzLjQgNDQuNSwxMTMuMSA0NC41LDY0LjEgNjIuMyw2NC4xIAoJCQk2Mi4zLDgyLjMgNzkuMSw3Mi43IDc5LjEsOS43IAkJIi8+CgkJPHBvbHlnb24gY2xhc3M9InN0MCIgcG9pbnRzPSI2Mi4zLDExMy4xIDEwNi45LDg3LjQgMTA2LjksODcuNCAxMDYuOSwyNS43IDkwLjEsMTYuMSA5MC4xLDc3LjggNjIuMyw5My44IAkJIi8+Cgk8L2c+CjwvZz4KPC9zdmc+Cg==)
[![Releases](https://img.shields.io/github/release/hashicorp/terraform-provider-kubernetes-alpha/all.svg?style=flat-square)](https://github.com/hashicorp/terraform-provider-kubernetes-alpha/releases)
[![LICENSE](https://img.shields.io/github/license/hashicorp/terraform-provider-kubernetes-alpha.svg?style=flat-square)](https://github.com/hashicorp/terraform-provider-kubernetes-alpha/blob/master/LICENSE)

A Terraform provider for Kubernetes which supports all API resources in a generic fashion.

This provider allows you to describe any Kubernetes resource using HCL. See [Moving from YAML to HCL](#moving-from-yaml-to-hcl) if you have YAML you want to use with the provider.

## ⚠️ Experimental status (please read)

This new provider represents a significant departure from the established practice of developing Terraform providers. It does not make use of the [Terraform Provider SDK](https://github.com/hashicorp/terraform-plugin-sdk) as all other providers currently do. This shift was necesary in order to leverege certain low-level features introduced in Terraform with version 0.12 which are currently not reflected in the SDK. Such features include a rich type system which allows for dynamically typed resource attributes. It was needed in order to implement variable-schema Kubernetes resources like Custom Resource Definitions. Once the new version of the SDK is available, this provider will need to be refactored significantly to bring it in line with the practises set by the SDK.

As a consequence of not using the provider SDK, certain "typical" features of Terraform such as planning changes have had to be partially reimplemented in the provider. The state of this implemtations is still evolving and as such may not yeld as smooth of an experince as other more mature providers. Particularly resource updating has rough edges that are being actively worked on.


Please regard this project as experimental. It still requires extensive testing and polishing to mature into production-ready quality.

**DO NOT USE IN PRODUCTION!**

Please file issues generously and detail your experience while using the provider. We encourage all types of feedback.

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

The `manifest` attribute of the `kubernetes_manifest` resource accepts any arbitrary Kubernetes API object, using Terraform's [map](https://www.terraform.io/docs/configuration/expressions.html#map) syntax. If you have YAML you want to use with this provider, we recommend that you convert it to a map as an initial step, rather than using `yamdecode` inside the resource block. 

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
