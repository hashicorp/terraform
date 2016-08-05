---
layout: "acme"
page_title: "Provider: ACME"
sidebar_current: "docs-acme-index"
description: |-
  The Automated Certificate Management Environment (ACME) provider is used to interact with an ACME Certificate Authority, such as Let's Encrypt (https://www.letsencrypt.org/). This provider can be used to both manage registrations and certificates.
---

# ACME Provider

The Automated Certificate Management Environment (ACME) provider is used to
interact with an ACME Certificate Authority, such as Let's Encrypt
(https://letsencrypt.org/). This provider can be used to both manage
registrations and certificates.

See the links on the sidebar for information on individual resource
configuraiton.

## About ACME

The Automated Certificate Management Environment (ACME) is an emerging
standard for the automation of a domain-validated certificate authority.
Clients set up **registrations** using a private key and contact information,
obtain **authorizations** for domains using a variety of challenges such as
HTTP, HTTPS (TLS), and DNS, with which they can request **certificates**. No
part of this process requires user interaction, a traditional blocker in
obtaining a domain validated certificate.

Currently the major ACME CA is Let's Encrypt (https://letsencrypt.org/),
but the ACME support in Terraform can be configured to use any ACME CA,
including an internal one that is set up using [Boulder][1].

You can read the ACME specification [here][2]. Note that the specification is
currently still in draft, and some features in the specification may not be
fully implemented in ACME CAs like Let's Encrypt or Boulder, and subsequently,
Terraform.

## Example Usage

The below example is an end-to-end demonstration of the setup of a basic
certificate, with a little help from the [`tls_private_key`][3] resource:

```
# Create the private key for the registration (not the certificate)
resource "tls_private_key" "private_key" {
  algorithm = "RSA"
}

# Set up a registration using a private key from tls_private_key
resource "acme_registration" "reg" {
  server_url      = "https://acme-staging.api.letsencrypt.org/directory"
  account_key_pem = "${tls_private_key.private_key.private_key_pem}"
  email_address   = "nobody@example.com"
}

# Create a certificate
resource "acme_certificate" "certificate" {
  server_url                = "https://acme-staging.api.letsencrypt.org/directory"
  account_key_pem           = "${tls_private_key.private_key.private_key_pem}"
  common_name               = "www.example.com"
  subject_alternative_names = ["www2.example.com"]

  dns_challenge {
    provider = "route53"
  }

  registration_url = "${acme_registration.reg.id}"
}
```

## Registration Credentials

Note that in the above usage example, `server_url` and `account_key_pem` are
required in both resources, and are not configured in a `provider` block.
This allows Terraform the freedom to set up a registration from scratch, with
nothing needing to be done out-of-band - as seen in the example above, the
`account_key_pem` is derived from a [`tls_private_key`][3] resource.

This also means that the two resources can be de-coupled from each other -
there is no need for `acme_registration` or `acme_certificate` to appear in
the same Terraform configuration. One configuration can set up the
registration, with another setting up the certificate, using the registration
from the previous configuration, or one supplied out-of-band.


[1]: https://github.com/letsencrypt/boulder
[2]: https://github.com/ietf-wg-acme/acme
[3]: /docs/providers/tls/index.html
