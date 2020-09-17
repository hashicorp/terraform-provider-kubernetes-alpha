variable "server_side_planning" {
  type    = bool
  default = true
}

variable "location" {
  type    = string
  default = "West Europe"
}


/*
  This creates a simple cluster on AKS.
*/
provider "azurerm" {
  version = ">=2.20.0"
  features {}
}

module "cluster" {
  source   = "./cluster"
  location = var.location
}


/*
  Here we create the Kubernetes resources on the AKS cluster.
  
  IMPORTANT: there is no explicit or implicit way to express dependency 
  of the Kubernetes resource on the AKS resource being present.
  You must split the apply into two operations. See README.md
*/

provider "kubernetes-alpha" {
  server_side_planning = var.server_side_planning

  host                   = module.cluster.host
  cluster_ca_certificate = module.cluster.cluster_ca_certificate
  client_certificate     = module.cluster.client_certificate
  client_key             = module.cluster.client_key
}

module "manifests" {
  source       = "./manifests"
  cluster_name = module.cluster.cluster_name
}
