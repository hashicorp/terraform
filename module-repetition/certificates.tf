
# This example is a little contrived in order to allow using a provider that
# only works locally and doesn't require any real credentials. We're using
# the TLS provider to generate certificates only as an example use-case to
# demonstrate the module for_each behavior. Please customize this example to
# try out other use-cases and other providers!
#
# This is _not_ intended as an example of a good way to generate TLS
# certificates in Terraform for production use!

module "ca" {
  source = "./certificate-authority"

  common_name       = "AwesomeCA"
  organization_name = "AwesomeCA"
}

resource "tls_private_key" "example_dot_com" {
  algorithm = "RSA"
  rsa_bits  = 2048
}

resource "tls_private_key" "example_dot_net" {
  algorithm = "RSA"
  rsa_bits  = 2048
}

locals {
  sites = {
    "example.com" = {
      dns_names = ["example.com", "*.example.com"]
      tls_key   = tls_private_key.example_dot_com
      org_name  = "Example Dot Com, Inc"
    }
    "example.net" = {
      dns_names = ["example.net", "*.example.net"]
      tls_key   = tls_private_key.example_dot_net
      org_name  = "Example.net, EMEA"
    }
  }
}

module "certificates" {
  source   = "./certificate"
  for_each = local.sites

  ca                = module.ca
  dns_names         = each.value.dns_names
  key               = each.value.tls_key
  organization_name = each.value.org_name
}

output "certificates_pem" {
  value = { for k, m in module.certificates : k => m.certificate_pem }
}
