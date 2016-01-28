---
layout: "vault"
page_title: "Provider: Vault"
sidebar_current: "docs-vault-index"
description: |-
  The Vault provider is used to interact with HashiCorp's Vault. The provider needs to be configured with the proper credentials before it can be used.
---

# Vault Provider

The Vault provider is used to interact with [HashiCorp's Vault](https://www.vaultproject.io/). The provider needs to be configured with the proper credentials before it can be used.

Use the navigation to the left to read about the available resources.

## Example Usage

```
variable "vault_address" { }
variable "vault_token" { }

# Configure the Vault Provider
provider "vault" {
  address     = "${var.vault_address}"

  # authentication details
  auth_method = "token"
  auth_config = {
    token = "${var.vault_token}"
  }
}
```

## Argument Reference

The following arguments are used to configure the Vault Provider:

* `address` - (Required) The address of the Vault server. Can also be specified
  with the `VAULT_ADDR` environment variable.
* `ca_cert` - (Optional) Absolute path to a PEM-encoded CA cert file to use to verify
  the Vault server SSL certificate. Can also be specified with the
  `VAULT_CACERT` environment variable.
* `ca_path` - (Optional) Absolute path to a directory of PEM-encoded CA cert files to
  verify the Vault server SSL certificate. If `ca_cert` is specified, its
  value will take precedence. Can also be specified with the `VAULT_CAPATH`
  environment variable.
* `auth_method` - (Optional) Specifies an alternate authentication method be
  used. (Defaults to `token`.)
* `auth_config` - (Required) A set of Key/Value pairs for authenticating with
  the selected method. Details below under [Auth Config](#auth-config)
* `allow_unverified_ssl` - (Optional) Boolean that can be set to true to
  disable SSL certificate verification. This should be used with care as it
  could allow an attacker to intercept your auth token. If omitted, default
  value is false. Can also be specified with the
  `VAULT_SKIP_VERIFY` environment variable.

<a id="auth-config"></a>

## Auth Config

Depending on which `auth_method` is used, different arguments are available for
`auth_config`.

For the `token` authentication method, the following arguments apply:

* `token` - (Required) A token to authenticate with. Can also be specified via
  the `VAULT_TOKEN` environment variable.

For the `cert` authentication method, the following arguments apply:

* `client_cert` - (Required, String) Contents of a client certificate to use for
  authentication.
* `client_key` - (Required, String) Content of a client key to use for
  authentication.
* `mount` - (Optional) Mount point of the authentication backend, defaults to
  `"cert"`.
