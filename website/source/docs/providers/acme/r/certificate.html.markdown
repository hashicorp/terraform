---
layout: "acme"
page_title: "ACME: certificate"
sidebar_current: "docs-acme-resource-certificate"
description: |-
  Creates and manages an ACME certificate.
---

# acme\_certificate

Use this resource to create and manage an ACME TLS certificate.

~> **NOTE:** Note that the example uses the
[Let's Encrypt staging environment][1]. If you are using Let's Encrypt, make
sure you change the URL to the correct endpoint (currently
`https://acme-v01.api.letsencrypt.org`).

~> **NOTE:** Some current ACME CA implementations like [Boulder][2] strip
most of the organization information out of a certificate request's subject,
so you may wish to confirm with the CA what behaviour to expect when using the
`certificate_request_pem` argument with this resource.

## Example

#### Full example with `common_name` and `subject_alternative_names` and DNS validation

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

#### Above example with HTTP/TLS validation

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

  http_challenge_port = 8080
  tls_challenge_port 8443 

  registration_url = "${acme_registration.reg.id}"
}
```

#### Full example with `certificate_request_pem` and DNS validation

```
resource "tls_private_key" "reg_private_key" {
  algorithm = "RSA"
}

resource "acme_registration" "reg" {
  server_url      = "https://acme-staging.api.letsencrypt.org/directory"
  account_key_pem = "${tls_private_key.private_key.private_key_pem}"
  email_address   = "nobody@example.com"
}

resource "tls_private_key" "cert_private_key" {
  algorithm = "RSA"
}

resource "tls_cert_request" "req" {
  key_algorithm   = "RSA"
  private_key_pem = "${tls_private_key.cert_private_key.private_key_pem}"
  dns_names       = ["www.example.com", "www2.example.com"]

  subject {
    common_name  = "www.example.com"
  }
}

resource "acme_certificate" "certificate" {
  server_url       = "https://acme-staging.api.letsencrypt.org/directory"
  account_key_pem  = "${tls_private_key.reg_private_key.private_key_pem}"
  certificate_request_pem = "${tls_cert_request.req.cert_request_pem}"

  dns_challenge {
    provider = "route53"
  }

  registration_url = "${acme_registration.reg.id}"
}
```

## Argument Reference

The resource takes the following arguments:

 * `server_url` (Required) - The URL of the ACME directory endpoint.
 * `account_key_pem` (Required) - The private key used to sign requests. This
    will be the private key that will be registered to the account.
 * `registration_url` (Required) - The URL that will be used to fetch the
   registrations's link to perform authorizations.
 * `common_name` - The certificate's common name, the primary domain that the 
   certificate will be recognized for. Required when not specifying a CSR.
 * `subject_alternative_names` - The certificate's subject alternative names,
   domains that this certificate will also be recognized for. Only valid when 
   not specifying a CSR.
 * `key_type` - The key type for the certificate's private key. Can be one of:
   `P256` and `P384` (for ECDSA keys of respective length) or `2048`, `4096`, 
   and `8192` (for RSA keys of respective length). Required when not
   specifying a CSR. The default is `2048` (RSA key of 2048 bits).
 * `certificate_request_pem` - A pre-created certificate request, such as one from
   [`tls_cert_request`][3], or one from an external source, in PEM format.
   Either this, or `common_name`, `key_type`, and optionally
   `subject_alternative_names` needs to be specified.
 * `min_days_remaining` (Optional) - The minimum amount of days remaining before the certificate
   expires before a renewal is attempted. The default is `7`. A value of less
   than 0 means that the certificate will never be renewed.
 * `dns_challenge` (Optional) - Select a [DNS challenge](#using-dns-challenges)
   to use in fulfilling the request. If this is used, HTTP and TLS challenges
   are disabled.
 * `http_challenge_port` (Optional) The port to use in the
   [HTTP challenge](#using-http-and-tls-challenges). Defaults to `80`.
 * `tls_challenge_port` (Optional) The port to use in the
   [TLS challenge](#using-http-and-tls-challenges). Defaults to `443`.

### Using DNS challenges

ACME and ACME CAs such as Let's Encrypt may support [DNS challenges][4], which
allows operators to respond to authorization challenges by provisioning a TXT
record on a specific domain.

Terraform, making use of [lego][5], responds to DNS challenges automatically
by utilizing one of lego's supported [DNS challenge providers][6]. Most
providers take credentials as environment variables, but if you would rather
use configuration for this purpose, you can through specifying `config` blocks
within a `dns_challenge` block, along with the `provider` parameter.

Example with Route 53 (AWS):

```
# Configure the AWS Provider
provider "aws" {
  access_key = "${var.aws_access_key}"
  secret_key = "${var.aws_secret_key}"
  region = "us-east-1"
}

# Create a certificate
resource "acme_certificate" "certificate" {
  ...

  dns_challenge {
    provider = "route53"
    config {
      AWS_ACCESS_KEY_ID     = "${var.aws_access_key}"
      AWS_SECRET_ACCESS_KEY = "${var.aws_secret_key}"
      AWS_DEFAULT_REGION    = "us-east-1"
    }
  }

  ...
}

```

### Using HTTP and TLS challenges

[HTTP challenges][7] and [TLS challenges][8] work via provisioning a response
message at a specific URL within a well known URI namespace on the hosts
being requested within a certificate.

This presents a unique challenge to Terraform, as normally, Terraform is more
than likely not being run from a live webserver. It is, however, possible to
proxy these requests to the host running Terraform. In order to do this,
perform the following:

 * Set your `http_challenge_port` or `tls_challenge_port` to non-standard
   ports, or leave them if you can assign the Terraform binary the
   `cap_net_bind_service=+ep` - (Linux hosts only).
   [Example configuration here.](#above-example-with-http-tls-validation)
 * Proxy the following to the host running Terraform, on the respective ports:
  * All requests on port 80 under the `/.well-known/acme-challenge/` URI
    namespace for HTTP challenges, or:
  * All TLS requests on port 443 for TLS challenges.

## Attribute Reference

The following attributes are exported:

 * `id` - The full URL of the certificate. Same as `certificate_url`.
 * `certificate_domain` - The common name of the certificate.
 * `certificate_url` - The URL for the certificate. Same as `id`.
 * `account_ref` - The URI of the registration account for this certificate.
   should be the same as `registration_url`.
 * `private_key_pem` - The certificate's private key, in PEM format, if the
   certificate was generated from scratch and not with `certificate_request_pem`. If
   `certificate_request_pem` was used, this will be blank.
 * `certificate_pem` - The certificate in PEM format.
 * `issuer_pem` - The intermediate certificate of the issuer.

[1]: https://letsencrypt.org/docs/staging-environment/
[2]: https://github.com/letsencrypt/boulder
[3]: /docs/providers/tls/r/cert_request.html
[4]: https://github.com/ietf-wg-acme/acme/blob/master/draft-ietf-acme-acme.md#dns
[5]: https://github.com/xenolf/lego
[6]: https://godoc.org/github.com/xenolf/lego/providers/dns
[7]: https://github.com/ietf-wg-acme/acme/blob/master/draft-ietf-acme-acme.md#http
[8]: https://github.com/ietf-wg-acme/acme/blob/master/draft-ietf-acme-acme.md#tls-with-server-name-indication-tls-sni
