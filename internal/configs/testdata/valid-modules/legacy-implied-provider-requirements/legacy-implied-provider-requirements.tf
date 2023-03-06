# Before Terraform v0.13 we didn't have any need for explicit provider
# requirements because there was only one namespace of providers.
#
# Now we have a heirarchical namespace for providers and so modules are
# supposed to declare specific source locations for every provider they use.
# However, as a special case we allow leaving undeclared any provider that was
# in the "hashicorp" namespace before Terraform v1.4, as a concession to
# backward compatibility with modules written for earlier versions of Terraform.
#
# The provider blocks below exercise all of the providers which this special
# behavior applies to. Terraform should infer a provider source address
# automatically for all of these, even though they aren't mentioned in a
# required_providers block. This test is here to make sure we don't
# unintentionally break our automatic inference for these particular provider
# names which existing modules might potentially be depending on implicitly.
#
# (The cutoff is at v1.4 rather than at v0.13 because versions between v0.13
# and v1.3 inclusive treated any undeclared dependency as an implied reference
# to an official provider. We tightened that to only a fixed set of already
# existing providers in v1.4 so that we can give better feedback when authors
# try use partner and community providers but do so incorrectly.)

resource "ad_thing" "example" {}
resource "archive_thing" "example" {}
resource "aws_thing" "example" {}
resource "awscc_thing" "example" {}
resource "azuread_thing" "example" {}
resource "azurerm_thing" "example" {}
resource "azurestack_thing" "example" {}
resource "boundary_thing" "example" {}
resource "cloudinit_thing" "example" {}
resource "consul_thing" "example" {}
resource "dns_thing" "example" {}
resource "external_thing" "example" {}
resource "google_thing" "example" {}
resource "googleworkspace_thing" "example" {}
resource "hashicups_thing" "example" {}
resource "hcp_thing" "example" {}
resource "hcs_thing" "example" {}
resource "helm_thing" "example" {}
resource "http_thing" "example" {}
resource "kubernetes_thing" "example" {}
resource "local_thing" "example" {}
resource "nomad_thing" "example" {}
resource "null_thing" "example" {}
resource "opc_thing" "example" {}
resource "oraclepaas_thing" "example" {}
resource "random_thing" "example" {}
resource "salesforce_thing" "example" {}
resource "template_thing" "example" {}
resource "terraform_thing" "example" {}
resource "tfcoremock_thing" "example" {}
resource "tfe_thing" "example" {}
resource "time_thing" "example" {}
resource "tls_thing" "example" {}
resource "vault_thing" "example" {}
resource "vsphere_thing" "example" {}
