# Releasing a New Version of the Protocol

Terraform's plugin protocol is the contract between Terraform's plugins and
Terraform, and as such releasing a new version requires some coordination
between those pieces. This document is intended to be a checklist to consult
when adding a new major version of the protocol (X in X.Y) to ensure that
everything that needs to be is aware of it.

## New Protobuf File

The protocol is defined in protobuf files that live in the hashicorp/terraform
repository. Adding a new version of the protocol involves creating a new
`.proto` file in that directory. It is recommended that you copy the latest
protocol file, and modify it accordingly.

## New terraform-plugin-go Package

The
[hashicorp/terraform-plugin-go](https://github.com/hashicorp/terraform-plugin-go)
repository serves as the foundation for Terraform's plugin ecosystem. It needs
to know about the new major protocol version. Either open an issue in that repo
to have the Plugin SDK team add the new package, or if you would like to
contribute it yourself, open a PR. It is recommended that you copy the package
for the latest protocol version and modify it accordingly.

## Update the Registry's List of Allowed Versions

The Terraform Registry validates the protocol versions a provider advertises
support for when ingesting providers. Providers will not be able to advertise
support for the new protocol version until it is added to that list.

## Update Terraform's Version Constraints

Terraform only downloads providers that speak protocol versions it is
compatible with from the Registry during `terraform init`. When adding support
for a new protocol, you need to tell Terraform it knows that protocol version.
Modify the `SupportedPluginProtocols` variable in hashicorp/terraform's
`internal/getproviders/registry_client.go` file to include the new protocol.

## Test Running a Provider With the Test Framework

Use the provider test framework to test a provider written with the new
protocol. This end-to-end test ensures that providers written with the new
protocol work correctly wtih the test framework, especially in communicating
the protocol version between the test framework and Terraform.

## Test Retrieving and Running a Provider From the Registry

Publish a provider, either to the public registry or to the staging registry,
and test running `terraform init` and `terraform apply`, along with exercising
any of the new functionality the protocol version introduces. This end-to-end
test ensures that all the pieces needing to be updated before practitioners can
use providers built with the new protocol have been updated.
