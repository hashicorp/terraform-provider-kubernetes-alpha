# Generic Terraform provider for Kubernetes (ALPHA)

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
