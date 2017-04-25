---
layout: "tls"
page_title: "TLS: tls_self_signed_cert"
sidebar_current: "docs-tls-resource-self-signed-cert"
description: |-
  Creates a self-signed TLS certificate in PEM format.
---

# tls\_self\_signed\_cert

Generates a *self-signed* TLS certificate in PEM format, which is the typical
format used to configure TLS server software.

Self-signed certificates are generally not trusted by client software such
as web browsers. Therefore clients are likely to generate trust warnings when
connecting to a server that has a self-signed certificate. Self-signed certificates
are usually used only in development environments or apps deployed internally
to an organization.

This resource is intended to be used in conjunction with a Terraform provider
that has a resource that requires a TLS certificate, such as:

* ``aws_iam_server_certificate`` to register certificates for use with AWS *Elastic
Load Balancer*, *Elastic Beanstalk*, *CloudFront* or *OpsWorks*.

* ``heroku_cert`` to register certificates for applications deployed on Heroku.

## Example Usage

```hcl
resource "tls_self_signed_cert" "example" {
  key_algorithm   = "ECDSA"
  private_key_pem = "${file(\"private_key.pem\")}"

  subject {
    common_name  = "example.com"
    organization = "ACME Examples, Inc"
  }

  validity_period_hours = 12

  allowed_uses = [
    "key_encipherment",
    "digital_signature",
    "server_auth",
  ]
}
```

## Argument Reference

The following arguments are supported:

* `key_algorithm` - (Required) The name of the algorithm for the key provided
  in `private_key_pem`.

* `private_key_pem` - (Required) PEM-encoded private key data. This can be
  read from a separate file using the ``file`` interpolation function. If the
  certificate is being generated to be used for a throwaway development
  environment or other non-critical application, the `tls_private_key` resource
  can be used to generate a TLS private key from within Terraform. Only
  an irreversable secure hash of the private key will be stored in the Terraform
  state.

* `subject` - (Required) The subject for which a certificate is being requested.
  This is a nested configuration block whose structure matches the
  corresponding block for [`tls_cert_request`](cert_request.html).

* `validity_period_hours` - (Required) The number of hours after initial issuing that the
  certificate will become invalid.

* `allowed_uses` - (Required) List of keywords each describing a use that is permitted
  for the issued certificate. The valid keywords are listed below.

* `dns_names` - (Optional) List of DNS names for which a certificate is being requested.

* `ip_addresses` - (Optional) List of IP addresses for which a certificate is being requested.

* `early_renewal_hours` - (Optional) If set, the resource will consider the certificate to
  have expired the given number of hours before its actual expiry time. This can be useful
  to deploy an updated certificate in advance of the expiration of the current certificate.
  Note however that the old certificate remains valid until its true expiration time, since
  this resource does not (and cannot) support certificate revocation. Note also that this
  advance update can only be performed should the Terraform configuration be applied during the
  early renewal period.

* `is_ca_certificate` - (Optional) Boolean controlling whether the CA flag will be set in the
  generated certificate. Defaults to `false`, meaning that the certificate does not represent
  a certificate authority.

The `allowed_uses` list accepts the following keywords, combining the set of flags defined by
both [Key Usage](https://tools.ietf.org/html/rfc5280#section-4.2.1.3) and
[Extended Key Usage](https://tools.ietf.org/html/rfc5280#section-4.2.1.12) in
[RFC5280](https://tools.ietf.org/html/rfc5280):

* `digital_signature`
* `content_commitment`
* `key_encipherment`
* `data_encipherment`
* `key_agreement`
* `cert_signing`
* `crl_signing`
* `encipher_only`
* `decipher_only`
* `any_extended`
* `server_auth`
* `client_auth`
* `code_signing`
* `email_protection`
* `ipsec_end_system`
* `ipsec_tunnel`
* `ipsec_user`
* `timestamping`
* `ocsp_signing`
* `microsoft_server_gated_crypto`
* `netscape_server_gated_crypto`

## Attributes Reference

The following attributes are exported:

* `cert_pem` - The certificate data in PEM format.
* `validity_start_time` - The time after which the certificate is valid, as an
  [RFC3339](https://tools.ietf.org/html/rfc3339) timestamp.
* `validity_end_time` - The time until which the certificate is invalid, as an
  [RFC3339](https://tools.ietf.org/html/rfc3339) timestamp.

## Automatic Renewal

This resource considers its instances to have been deleted after either their validity
periods ends or the early renewal period is reached. At this time, applying the
Terraform configuration will cause a new certificate to be generated for the instance.

Therefore in a development environment with frequent deployments it may be convenient
to set a relatively-short expiration time and use early renewal to automatically provision
a new certificate when the current one is about to expire.

The creation of a new certificate may of course cause dependent resources to be updated
or replaced, depending on the lifecycle rules applying to those resources.
