---
layout: "tls"
page_title: "TLS: tls_locally_signed_cert"
sidebar_current: "docs-tls-resourse-locally-signed-cert"
description: |-
  Creates a locally-signed TLS certificate in PEM format.
---

# tls\_locally\_signed\_cert

Generates a TLS ceritifcate using a *Certificate Signing Request* (CSR) and
signs it with a provided certificate authority (CA) private key.

Locally-signed certificates are generally only trusted by client software when
setup to use the provided CA. They are normally used in development environments
or when deployed internally to an organization.

## Example Usage

```
resource "tls_locally_signed_cert" "example" {
    cert_request_pem = "${file(\"cert_request.pem\")}"

    ca_key_algorithm = "ECDSA"
    ca_private_key_pem = "${file(\"ca_private_key.pem\")}"
    ca_cert_pem = "${file(\"ca_cert.pem\")}"

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

* `cert_request_pem` - (Required) PEM-encoded request certificate data.

* `ca_key_algorithm` - (Required) The name of the algorithm for the key provided
  in `ca_private_key_pem`.

* `ca_private_key_pem` - (Required) PEM-encoded private key data for the CA.
  This can be read from a separate file using the ``file`` interpolation
  function.

* `ca_cert_pem` - (Required) PEM-encoded certificate data for the CA.

* `validity_period_hours` - (Required) The number of hours after initial issuing that the
  certificate will become invalid.

* `allowed_uses` - (Required) List of keywords each describing a use that is permitted
  for the issued certificate. The valid keywords are listed below.

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
