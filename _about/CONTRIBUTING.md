## Building and installing

As we do not yet publish releases for this provider to registry.terraform.io, you have to either [download a release from Github](https://github.com/hashicorp/terraform-provider-kubernetes-alpha/releases) or build and install it manually as indicated below.

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

### Using `-plugin-dir` 

Alternative, you can run:

```
make build
```

This will place the provider binary in the top level of the provider directory. You can then use it with terraform by specifying the `-plugin-dir` option when running `terraform init`

```
terraform init -plugin-dir /path/to/terraform-provider-alpha
```