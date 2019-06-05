---
layout: "backend-types"
page_title: "Backend Type: pg"
sidebar_current: "docs-backends-types-standard-pg"
description: |-
  Terraform can store state remotely in a Postgres database with locking.
---

# pg

**Kind: Standard (with locking)**

Stores the state in a [Postgres database](https://www.postgresql.org) version 9.5 or newer.

This backend supports [state locking](/docs/state/locking.html).

## Example Configuration

```hcl
terraform {
  backend "pg" {
    conn_str = "postgres://user:pass@db.example.com/terraform_backend"
  }
}
```

Before initializing the backend with `terraform init`, the database must already exist:

```
createdb terraform_backend
```

This `createdb` command is found in [Postgres client applications](https://www.postgresql.org/docs/9.5/reference-client.html) which are installed along with the database server.

We recommend using a [partial configuration](/docs/backends/config.html#partial-configuration) for the `conn_str` variable, because it typically contains access credentials that should not be committed to source control:

```hcl
terraform {
  backend "pg" {}
}
```

Then, set the credentials when initializing the configuration:

```
terraform init -backend-config="conn_str=postgres://user:pass@db.example.com/terraform_backend"
```

To use a Postgres server running on the same machine as Terraform, configure localhost with SSL disabled:

```
terraform init -backend-config="conn_str=postgres://localhost/terraform_backend?sslmode=disable"
```

## Example Referencing

To make use of the pg remote state we can use the [`terraform_remote_state` data source](/docs/providers/terraform/d/remote_state.html).

```hcl
data "terraform_remote_state" "network" {
  backend = "pg"
  config {
    conn_str = "postgres://localhost/terraform_backend"
  }
}
```

## Configuration Variables

The following configuration options or environment variables are supported:

 * `conn_str` - (Required) Postgres connection string; a `postgres://` URL
 * `schema_name` - Name of the automatically-managed Postgres schema, default `terraform_remote_state`.
 * `skip_schema_creation` - If set to `true`, the Postgres schema must already exist. Terraform won't try to create the schema. Useful when the Postgres user does not have "create schema" permission on the database.

## Technical Design

Postgres version 9.5 or newer is required to support advisory locks and the "ON CONFLICT" upsert syntax.

This backend creates one table **states** in the automatically-managed Postgres schema configured by the `schema_name` variable.

The table is keyed by the [workspace](/docs/state/workspaces.html) name. If workspaces are not in use, the name `default` is used.

Locking is supported using [Postgres advisory locks](https://www.postgresql.org/docs/9.5/explicit-locking.html#ADVISORY-LOCKS). [`force-unlock`](https://www.terraform.io/docs/commands/force-unlock.html) is not supported, because these database-native locks will automatically unlock when the session is aborted or the connection fails. To see outstanding locks in a Postgres server, use the [`pg_locks` system view](https://www.postgresql.org/docs/9.5/view-pg-locks.html).

The **states** table contains:

 * a serial integer `id`, used as the key for advisory locks
 * the workspace `name` key as *text* with a unique index
 * the Terraform state `data` as *text*
