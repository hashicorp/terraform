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
    conn_str = "postgres://localhost/terraform_backend"
  }
}
```

Before initializing the backend with `terraform init`, the database must already exist.

We recommend using a [partial configuration](/docs/backends/config.html#partial-configuration) for the `conn_str` variable, because it typically contains access credentials that should not be committed to source control.

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
 * `lock` - Use locks to synchronize state access, default `true`
 * `schema_name` - Name of the automatically-managed Postgres schema to store locks & state, default `terraform_remote_backend`.
 
## Technical Design

Postgres version 9.5 or newer is required to support the "ON CONFLICT" upsert syntax and *jsonb* data type.

This backend creates two tables, **states** and **locks**, in the automatically-managed Postgres schema configured by the `schema_name` variable.

Both tables are keyed by the [workspace](/docs/state/workspaces.html) name. If workspaces are not in use, the name `default` is used.

The **states** table contains:

 * the workspace `name` key as *text* with a unique index
 * the Terraform state `data` JSON as *text*.

The **locks** table contains:

 * the workspace `name` key as *text* with a unique index
 * the lock's `info` JSON as *jsonb*.
