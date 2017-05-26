---
layout: "vault"
page_title: "Provider: Vault"
sidebar_current: "docs-vault-index"
description: |-
  The Vault provider allows Terraform to read from, write to, and configure Hashicorp Vault
---

# Vault Provider

The Vault provider allows Terraform to read from, write to, and configure
[Hashicorp Vault](https://vaultproject.io/).

~> **Important** Interacting with Vault from Terraform causes any secrets
that you read and write to be persisted in both Terraform's state file
*and* in any generated plan files. For any Terraform module that reads or
writes Vault secrets, these files should be treated as sensitive and
protected accordingly.

This provider serves two pretty-distinct use-cases, which each have their
own security trade-offs and caveats that are covered in the sections that
follow. Consider these carefully before using this provider within your
Terraform configuration.

## Configuring and Populating Vault

Terraform can be used by the Vault adminstrators to configure Vault and
populate it with secrets. In this case, the state and any plans associated
with the configuration must be stored and communicated with care, since they
will contain in cleartext any values that were written into Vault.

Currently Terraform has no mechanism to redact or protect secrets
that are provided via configuration, so teams choosing to use Terraform
for populating Vault secrets should pay careful attention to the notes
on each resource's documentation page about how any secrets are persisted
to the state and consider carefully whether such usage is compatible with
their security policies.

Except as otherwise noted, the resources that write secrets into Vault are
designed such that they require only the *create* and *update* capabilities
on the relevant resources, so that distinct tokens can be used for reading
vs. writing and thus limit the exposure of a compromised token.

## Using Vault credentials in Terraform configuration

Most Terraform providers require credentials to interact with a third-party
service that they wrap. This provider allows such credentials to be obtained
from Vault, which means that operators or systems running Terraform need
only access to a suitably-privileged Vault token in order to temporarily
lease the credentials for other providers.

Currently Terraform has no mechanism to redact or protect secrets that
are returned via data sources, so secrets read via this provider will be
persisted into the Terraform state, into any plan files, and in some cases
in the console output produced while planning and applying. These artifacts
must therefore all be protected accordingly.

To reduce the exposure of such secrets, the provider requests a Vault token
with a relatively-short TTL (20 minutes, by default) which in turn means
that where possible Vault will revoke any issued credentials after that
time, but in particular it is unable to retract any static secrets such as
those stored in Vault's "generic" secret backend.

The requested token TTL can be controlled by the `max_lease_ttl_seconds`
provider argument described below. It is important to consider that Terraform
reads from data sources during the `plan` phase and writes the result into
the plan. Thus a subsequent `apply` will likely fail if it is run after the
intermediate token has expired, due to the revocation of the secrets that
are stored in the plan.

Except as otherwise noted, the resources that read secrets from Vault
are designed such that they require only the *read* capability on the relevant
resources.

## Provider Arguments

The provider configuration block accepts the following arguments.
In most cases it is recommended to set them via the indicated environment
variables in order to keep credential information out of the configuration.

* `address` - (Required) Origin URL of the Vault server. This is a URL
  with a scheme, a hostname and a port but with no path. May be set
  via the `VAULT_ADDR` environment variable.

* `token` - (Required) Vault token that will be used by Terraform to
  authenticate. May be set via the `VAULT_TOKEN` environment variable.
  If none is otherwise supplied, Terraform will attempt to read it from
  `~/.vault-token` (where the vault command stores its current token).
  Terraform will issue itself a new token that is a child of the one given,
  with a short TTL to limit the exposure of any requested secrets.

* `ca_cert_file` - (Optional) Path to a file on local disk that will be
  used to validate the certificate presented by the Vault server.
  May be set via the `VAULT_CACERT` environment variable.

* `ca_cert_dir` - (Optional) Path to a directory on local disk that
  contains one or more certificate files that will be used to validate
  the certificate presented by the Vault server. May be set via the
  `VAULT_CAPATH` environment variable.

* `client_auth` - (Optional) A configuration block, described below, that
  provides credentials used by Terraform to authenticate with the Vault
  server. At present there is little reason to set this, because Terraform
  does not support the TLS certificate authentication mechanism.

* `skip_tls_verify` - (Optional) Set this to `true` to disable verification
  of the Vault server's TLS certificate. This is strongly discouraged except
  in prototype or development environments, since it exposes the possibility
  that Terraform can be tricked into writing secrets to a server controlled
  by an intruder. May be set via the `VAULT_SKIP_VERIFY` environment variable.

* `max_lease_ttl_seconds` - (Optional) Used as the duration for the
  intermediate Vault token Terraform issues itself, which in turn limits
  the duration of secret leases issued by Vault. Defaults to 20 minutes
  and may be set via the `TERRAFORM_VAULT_MAX_TTL` environment variable.
  See the section above on *Using Vault credentials in Terraform configuration*
  for the implications of this setting.

The `client_auth` configuration block accepts the following arguments:

* `cert_file` - (Required) Path to a file on local disk that contains the
  PEM-encoded certificate to present to the server.

* `key_file` - (Required) Path to a file on local disk that contains the
  PEM-encoded private key for which the authentication certificate was issued.

## Example Usage

```hcl
provider "vault" {
  # It is strongly recommended to configure this provider through the
  # environment variables described above, so that each user can have
  # separate credentials set in the environment.
  #
  # This will default to using $VAULT_ADDR
  # But can be set explicitly
  # address = "https://vault.example.net:8200"
}

resource "vault_generic_secret" "example" {
  path = "secret/foo"

  data_json = <<EOT
{
  "foo":   "bar",
  "pizza": "cheese"
}
EOT
}
```

