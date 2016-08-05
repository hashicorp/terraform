---
layout: "acme"
page_title: "ACME: registration"
sidebar_current: "docs-acme-resource-registration"
description: |-
  Creates and manages an ACME registration.
---

# acme\_registration

Use this resource to create and manage an ACME registration.

~> **NOTE:** Note that the example uses the
[Let's Encrypt staging environment][1]. If you are using Let's Encrypt, make
sure you change the URL to the correct endpoint (currently
`https://acme-v01.api.letsencrypt.org`).

~> **NOTE:** While the ACME draft does contain provisions for deactivating
registrations, implementation is still in development, so if this resource in
Terraform is destroyed, the registration is not completely deleted.

## Example

The following creates an ACME registration off of a private key generated with
the [`tls_private_key`][2] resource.

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
```

## Argument Reference

The resource takes the following arguments:

 * `server_url` (Required) - The URL of the ACME directory endpoint.
 * `account_key_pem` (Required) - The private key used to sign requests. This
    is the private key that will be registered to the account.
 * `email_address` (Required) - The email address that will be attached as a
   contact to the registration.

## Attribute Reference

The following attributes are exported:

 * `id` - The full URL of the registration. Same as `registration_url`.
 * `registration_body`: The raw body of the registration response, in JSON
   format.
 * `registration_url`: The full URL of the registration. Same as `id`.
 * `registration_new_authz_url`: The full URL to the endpoint used to create
   new authorizations.
 * `registration_tos_url`: The full URL to the CA's terms of service.

[1]: https://letsencrypt.org/docs/staging-environment/
[2]: /docs/providers/tls/index.html
