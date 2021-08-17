---
layout: "language"
page_title: "Boundary - Configuration Language"
sidebar_current: "docs-config-boundary"
description: |-
  Boundary connections can be used to access infrastructure in a Zero Trust Network.
---

# Boundary

[Boundary](https://boundaryproject.io) is an identity aware proxy that
authenticate and authorize access to remote infrastructure. Using Boundary with
Terraform makes it possible to manage infrastructure that is not directly
connected to internet.

~> Warning: This feature is an experiment and may have breaking changes in
minor or patch releases.


## Using Boundary connections

A Boundary connection is created via a `boundary` block in the root module:

```hcl
terraform {
  # the 'boundary' experiment must be enabled to use this feature
  experiments = [ boundary ]

  required_providers {
    consul = {
      source  = "hashicorp/consul"
      version = "2.12.0"
    }
    vault = {
      source  = "hashicorp/vault"
      version = "2.21.0"
    }
  }
}

boundary {
  address = "https://boundary.hashicorp.com:9200"

  connection "consul" {
    target_scope_name = "Services"
    target_name       = "consul"
    listen_port       = 8500
  }

  connection "vault" {
    target_scope_name = "Services"
    target_name       = "vault"
    listen_port       = 8200
  }
}

provider "vault" {
  address         = "https://${boundary.vault.listen_addr}:${boundary.vault.listen_port}"
  skip_tls_verify = true
}

provider "consul" {
  address        = "https://${boundary.consul.listen_addr}:${boundary.consul.listen_port}"
  insecure_https = true
}
```

Once a Boundary is connected, you can reference it in expressions as
`boundary.<NAME>`.


## Using Boundary credential brokering

Credentials brokering in Boundary can be used to dynamicly generate credentials
for a connection. When credential brokering is enable for a target, the
credentials are available in expressions in the `boundary.<NAME>.credentials` list:

```hcl
terraform {
  experiments = [boundary]

  required_providers {
    postgresql = {
      source  = "cyrilgdn/postgresql"
      version = "1.13.0"
    }
  }
}

locals {
  foo = "1234"
}

boundary {
  address = "https://boundary.hashicorp.com:9200"

  connection "postgres" {
    target_name       = "default.db"
    target_scope_name = "Databases"
    listen_port       = 5432
  }
}

provider "postgresql" {
  port     = boundary.postgres.listen_port
  username = boundary.postgres.credentials[0].username
  password = boundary.postgres.credentials[0].password
}
```

## Boundary Arguments

Each Boundary blocks configures the client used to connect to the Boundary
service and one or more connection. The `boundary` block accepts the following
arguments:

- `address` (string) - Address of the Boundary controller, as a complete URL (e.g. https://boundary.example.com:9200). This can also be specified via the BOUNDARY_ADDR environment variable.
- `ca_cert` (string) - Path on the local disk to a single PEM-encoded CA certificate to verify the Controller or Worker's server's SSL certificate. This takes precedence over `ca_path`. This can also be specified via the `BOUNDARY_CACERT` environment variable.
- `ca_path` (string) - Path on the local disk to a directory of PEM-encoded CA certificates to verify the SSL certificate of the Controller. This can also be specified via the `BOUNDARY_CAPATH` environment variable.
- `client_cert` (string) - Path on the local disk to a single PEM-encoded CA certificate to use for TLS authentication to the Boundary Controller. If this argument is specified, client_key is also required. This can also be specified via the `BOUNDARY_CLIENT_CERT` environment variable.
- `client_key` (string) - Path on the local disk to a single PEM-encoded private key matching the client certificate from `client_cert`. This can also be specified via the `BOUNDARY_CLIENT_KEY` environment variable.
- `tls_insecure` (bool) - Disable verification of TLS certificates. Using this option is highly discouraged as it decreases the security of data transmissions to and from the Boundary server. The default is `false`. This can also be specified via the `BOUNDARY_TLS_INSECURE` environment variable.
- `tls_server_name` (string) - Name to use as the SNI host when connecting to the Boundary server via TLS. This can also be specified via the `BOUNDARY_TLS_SERVER_NAME` environment variable.
- `token` (string) - If specified, the given value will be used as the token for the call. Overrides the `token_name` parameter. This can also be specified via the `BOUNDARY_TOKEN` environment variable.
- `keyring_type` (string) - The type of keyring to use. Defaults to `auto` which will use the Windows credential manager, OSX keychain, or cross-platform password store depending on platform. Set to `none` to disable keyring functionality. Available types, depending on platform, are: `wincred`, `keychain`, `pass`, and `secret-service`. This can also be specified via the `BOUNDARY_KEYRING_TYPE` environment variable.
- `token_name` (string) - If specified, the given value will be used as the name when storing the token in the system credential store. This can allow switching user identities for different commands. This can also be specified via the `BOUNDARY_TOKEN_NAME` environment variable.

Each connection block accepts the following arguments:

- `authorization_token` (string) - Only needed if `target_id` is not set. The authorization string returned from the Boundary controller via an `authorize-session` action against a target. This can also be specified via the `BOUNDARY_CONNECT_AUTHZ_TOKEN` environment variable.
- `listen_addr` (string) - If set, the proxy will attempt to bind its listening address to the given value, which must be an IP address. If it cannot, Terraform will error. If not set, defaults to the most common IPv4 loopback address (127.0.0.1). This can also be specified via the `BOUNDARY_CONNECT_LISTEN_ADDR` environment variable.
- `listen_port` (number) - The proxy will attempt to bind its listening port to the given value. If it cannot, Terraform error. This can also be specified via the `BOUNDARY_CONNECT_LISTEN_PORT` environment variable.
- `host_id` (string) - The ID of a specific host to connect to out of the hosts from the target's host sets. If not specified, one is chosen at random.
- `target_id` (string) - The ID of the target to authorize against. Cannot be used with `authorization_token`.
- `target_name` (string) - Target name, if authorizing the session via scope parameters and target name.
- `target_scope_id` (string) - Target scope ID, if authorizing the session via scope parameters and target name. Mutually exclusive with `target_scope_name`. This can also be specified via the `BOUNDARY_CONNECT_TARGET_SCOPE_ID` environment variable.
- `target_scope_name` (string) - Target scope name, if authorizing the session via scope parameters and target name. Mutually exclusive with `target_scope_id`. This can also be specified via the `BOUNDARY_CONNECT_TARGET_SCOPE_NAME` environment variable.
