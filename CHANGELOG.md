## 0.3.0 (Unreleased)

FEATURES
* provider can now manage Custom Resources as per Kinds installed by their parent CRDs
* uses OpenAPI defitions from the target cluster to enforce resource structure and attribute types

ENHANCEMENTS
* completely refactored on top of the [terraform-plugin-go](https://github.com/hashicorp/terraform-plugin-go) SDK
* validations have been added to ensure manifests only specify a namespace when appropriate

DEPRECATIONS:
* the `server_side_planning` configuration attribute has been removed. All planning now uses the OpenAPI based mechanism.

## 0.2.1 (September 18, 2020)

FEATURES
* Add support for waiting on resource fields to reach a certain value (#105)
* Add standalone process debug mode (#121)

BUG FIXES
* Fix login with username and password (#113)
* Fix acceptance tests to work with terraform 0.13
* Defer client initialisation to better cope with transient incomplete client configuration

## 0.2.0 (August 26, 2020)

FEATURES
  * Add wait_for block to kubernetes_manifest resource (#95)

ENHANCEMENTS
  * Published to the Terraform registry

BUG FIXES

## 0.1.0 (June 26, 2020)
