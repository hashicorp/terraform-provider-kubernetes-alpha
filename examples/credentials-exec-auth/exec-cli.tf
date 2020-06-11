# This examples demonstrate how to configure the provider for accessing an EKS cluster using credentials dispatched by the AWS cli.
#
# It is based on the instructions described here: https://docs.aws.amazon.com/eks/latest/userguide/create-kubeconfig.html#create-kubeconfig-manually
#
# Requirement: the AWS cli tool should be installed and confirmed working.
# If it is not accesible in the PATH, the full path to the 'aws' tool should be used in the "command" attribute.

provider "kubernetes-alpha" {
  host = "https://84FEE4598F246F88925FC81F77877F99.yl4.eu-central-1.eks.amazonaws.com"

  # Cluster CA certificate obtained from EKS
  cluster_ca_certificate = base64decode("LS0tLS1CRUdJTiBDRVJUSUZJQ0FURS0tLS0tCk1JSUN5RENDQWJDZ0F3SUJBZ0lCQURBTkJna3Foa2lHOXcwQkFRc0ZBREFWTVJNd0VRWURWUVFERXdwcmRXSmwKY201bGRHVnpNQjRYRFRJd01EWXdPVEUxTkRrek1Wb1hEVE13TURZd056RTFORGt6TVZvd0ZURVRNQkVHQTFVRQpBeE1LYTNWaVpYSnVaWFJsY3pDQ0FTSXdEUVlKS29aSWh2Y05BUUVCQlFBRGdnRVBBRENDQVFvQ2dnRUJBTU02CkdaWUhVZjB1RStuRGh6bFExWE5wMFJWVlRxTlVUYzY2VjJrWkJkNHlqbWR1VmNLNG5FZzhYR2kxY3ZWMVFSSjIKcTlqR3dzdXNTMjRsRjh6QkNkMVJXakN1U3BwSThQbVk3TFBiOHV5UkRaMlRnSWl5TGNkSW1IU21xMlZ1WGxqcAp5V08zcVg2WnkrUlVWRWRNUW1abEEvcWdSRjlLTVZIcDcwWk95VmdDSnJhM2dTMnJWUUovVFRZaWVOb3ovUGFkCmladzdxWnlnS3RaejByZnpLdm1TQUhpZG5OeDVNanh6elJJWkQvbk9QbjNKelpPQWFkL2VMM0dXR0hPdFhiV1kKb3VOT1gxVklOMFBSWjdWRDc5eVlSMVBsZ2hNZXhRQ055M0FKT1dOOUFMakZqRFErN0xWS2o2YjBaS3V2L3Azawp4QXJJR1JkNENuZjlDNEJOVEpVQ0F3RUFBYU1qTUNFd0RnWURWUjBQQVFIL0JBUURBZ0trTUE4R0ExVWRFd0VCCi93UUZNQU1CQWY4d0RRWUpLb1pJaHZjTkFRRUxCUUFEZ2dFQkFKOC81Qklva0pyNEx4MVhKQXlsc25ySkhISGMKeTlIck9lU3lFTW9xNGRCcm93c2FKaWF6cWxPY2RqV2RjUmVaVkRPWkZRaDRJTE1NYlQ3VmhvRU01bXZsYnQyWAptYkF0RkNQVG01ZFBtbDVFSUI4MEsrWmh2MHA5bE1JMDRwT1lKYnlHblJpc3kvdGxjUk9YQVFQam5hQ0w0Z09QCjZhZFU2dEVrbmV0M3NjZHpPU21ZeUpWZ0xCTG1UOTZLV1RaOGRDQWtrazhrVE5FUWZ4WWdKNEUvOG1kaWpVVzkKbTN3SW9VaHVpT0tLSWxaOGJkeXNoVVhaL3NndlZWYnpnZWZYbXlIZS9sRWZ4RzE0LzJSOSs2cWZOSS95cHFRVwoya253WmtCWHIrWWhZTzRrUUFwMUVmK1BrOHc1VHRLb2dDZndIQ25XcHk5SGFacEMyc1N4dmNLSzEyST0KLS0tLS1FTkQgQ0VSVElGSUNBVEUtLS0tLQo=")

  exec = {
    api_version = "client.authentication.k8s.io/v1alpha1"

    command = "aws" # this is the actual 'aws' cli tool

    args = ["--region", "eu-central-1", "eks", "get-token", "--cluster-name", "my-eks-cluster"]

    env = {
      # Credentials for the AWS cli tool
      "AWS_PROFILE"           = "hashicorp"
      # "AWS_ACCESS_KEY_ID"     = "my-access-key-id"
      # "AWS_SECRET_ACCESS_KEY" = "my-secret-key-data"
    }
  }
}

resource "kubernetes_manifest" "test-namespace" {
  provider = kubernetes-alpha

  manifest = {
    "apiVersion" = "v1"
    "kind"       = "Namespace"
    "metadata" = {
      "name" = "tf-demo"
    }
  }
}