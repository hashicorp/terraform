---
layout: "tls"
page_title: "Provider: TLS"
sidebar_current: "docs-tls-index"
description: |-
  The TLS provider provides utilities for working with Transport Layer Security keys and certificates.
---

# TLS Provider

The TLS provider provides utilities for working with *Transport Layer Security*
keys and certificates. It provides resources that
allow private keys, certificates and certficate requests to be
created as part of a Terraform deployment.

Another name for Transport Layer Security is *Secure Sockets Layer*,
or SSL. TLS and SSL are equivalent when considering the resources
managed by this provider.

This provider is not particularly useful on its own, but it can be
used to create certificates and credentials that can then be used
with other providers when creating resources that expose TLS
services or that themselves provision TLS certificates.

Use the navigation to the left to read about the available resources.

## Example Usage

```
## This example create a self-signed certificate for a development
## environment.
## THIS IS NOT RECOMMENDED FOR PRODUCTION SERVICES.
## See the detailed documentation of each resource for further
## security considerations and other practical tradeoffs.

resource "tls_private_key" "example" {
    algorithm = "ECDSA"
}

resource "tls_self_signed_cert" "example" {
    key_algorithm = "${tls_private_key.example.algorithm}"
    private_key_pem = "${tls_private_key.example.private_key_pem}"

    # Certificate expires after 12 hours.
    validity_period_hours = 12

    # Generate a new certificate if Terraform is run within three
    # hours of the certificate's expiration time.
    early_renewal_hours = 3

    # Reasonable set of uses for a server SSL certificate.
    allowed_uses = [
        "key_encipherment",
        "digital_signature",
        "server_auth",
    ]

    dns_names = ["example.com", "example.net"]

    subject {
        common_name = "example.com"
        organization = "ACME Examples, Inc"
    }
}

# For example, this can be used to populate an AWS IAM server certificate.
resource "aws_iam_server_certificate" "example" {
    name = "example_self_signed_cert"
    certificate_body = "${tls_self_signed_cert.example.cert_pem}"
    private_key = "${tls_private_key.example.private_key_pem}"
}
```
