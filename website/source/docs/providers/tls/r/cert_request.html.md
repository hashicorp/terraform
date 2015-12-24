---
layout: "tls"
page_title: "TLS: tls_cert_request"
sidebar_current: "docs-tls-resourse-cert-request"
description: |-
  Creates a PEM-encoded certificate request.
---

# tls\_cert\_request

Generates a *Certificate Signing Request* (CSR) in PEM format, which is the
typical format used to request a certificate from a certificate authority.

This resource is intended to be used in conjunction with a Terraform provider
for a particular certificate authority in order to provision a new certificate.
This is a *logical resource*, so it contributes only to the current Terraform
state and does not create any external managed resources.

## Example Usage

```
resource "tls_cert_request" "example" {
    key_algorithm = "ECDSA"
    private_key_pem = "${file(\"private_key.pem\")}"

    subject {
        common_name = "example.com"
        organization = "ACME Examples, Inc"
    }
}
```

## Argument Reference

The following arguments are supported:

* `key_algorithm` - (Required) The name of the algorithm for the key provided
in `private_key_pem`.

* `private_key_pem` - (Required) PEM-encoded private key data. This can be
read from a separate file using the ``file`` interpolation function. Only
an irreversable secure hash of the private key will be stored in the Terraform
state.

* `subject` - (Required) The subject for which a certificate is being requested. This is
a nested configuration block whose structure is described below.

* `dns_names` - (Optional) List of DNS names for which a certificate is being requested.

* `ip_addresses` - (Optional) List of IP addresses for which a certificate is being requested.

The nested `subject` block accepts the following arguments, all optional, with their meaning
corresponding to the similarly-named attributes defined in
[RFC5290](https://tools.ietf.org/html/rfc5280#section-4.1.2.4):

* `common_name` (string)

* `organization` (string)

* `organizational_unit` (string)

* `street_address` (list of strings)

* `locality` (string)

* `province` (string)

* `country` (string)

* `postal_code` (string)

* `serial_number` (string)

## Attributes Reference

The following attributes are exported:

* `cert_request_pem` - The certificate request data in PEM format.
