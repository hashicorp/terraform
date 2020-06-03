
resource "tls_private_key" "ca" {
  # Please note that generating private keys like this in Terraform will cause
  # the secret key material to be recorded in the Terraform state, which in a
  # production system would usually require that state to be saved in a secure
  # location.

  algorithm = "RSA"
  rsa_bits  = 2048
}

resource "tls_self_signed_cert" "ca" {
  key_algorithm   = tls_private_key.ca.algorithm
  private_key_pem = tls_private_key.ca.private_key_pem

  is_ca_certificate = true
  allowed_uses      = ["cert_signing"]

  validity_period_hours = 24

  subject {
    common_name  = var.common_name
    organization = var.organization_name
  }
}
