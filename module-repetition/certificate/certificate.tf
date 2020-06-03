
# As is true for the rest of this example, this is _not_ intended as a good
# example of how to generate TLS certificates in Terraform. It's just a
# contrived example showing how to use module for_each. In particular, it's
# _very_ weird for the consumer of a certificate to be the one responsible for
# signing its cert request using the CA key!

resource "tls_cert_request" "cert" {
  key_algorithm   = var.key.algorithm
  private_key_pem = var.key.private_key_pem

  dns_names = var.dns_names

  subject {
    common_name  = var.dns_names[0]
    organization = var.organization_name
  }
}

resource "tls_locally_signed_cert" "cert" {
  cert_request_pem = tls_cert_request.cert.cert_request_pem

  ca_key_algorithm   = var.ca.key_algorithm
  ca_private_key_pem = var.ca.private_key_pem
  ca_cert_pem        = var.ca.cert_pem

  validity_period_hours = 24
  allowed_uses = [
    "server_auth",
  ]
}
