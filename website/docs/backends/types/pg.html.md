---
layout: "backend-types"
page_title: "Backend Type: pg"
sidebar_current: "docs-backends-types-standard-pg"
description: |-
  Terraform can store state remotely in a Postgres database with locking.
---

# pg

**Kind: Standard with locking**

Stores the state in a [Postgres database](https://www.postgresql.org) using two tables, `states` and `locks`.

Supports Postgres version 10 and newer.

## Example Configuration

```hcl
terraform {
  backend "pg" {
    conn_str = "postgres://localhost/terraform_backend"
  }
}
```

This assumes we have a database created called `terraform_backend`. The
Terraform state is stored in a self-managed schema, by default 
`terraform_remote_backend`.

Note that for the access credentials we recommend using a
[partial configuration](/docs/backends/config.html).

## Using the pg remote state

To make use of the pg remote state we can use the
[`terraform_remote_state` data
source](/docs/providers/terraform/d/remote_state.html).

```hcl
data "terraform_remote_state" "network" {
  backend = "pg"
  config {
    conn_str = "postgres://localhost/terraform_backend"
  }
}
```

The `terraform_remote_state` data source will return all of the root module 
outputs defined in the referenced remote state (but not any outputs from 
nested modules unless they are explicitly output again in the root). An 
example output might look like:

```
data.terraform_remote_state.network:
  id = 2018-10-11 01:57:59.780010914 +0000 UTC
  backend = pg
  config.% = 1
  config.conn_str = postgres://localhost/terraform_backend
```

## Configuration variables

The following configuration options or environment variables are supported:

 * `conn_str` - (Required) Postgres connection string; a `postgres://` URL
 * `lock` - Use locks to synchronize state access, default `true`
 * `schema_name` - Name of the automatically managed Postgres schema to store locks & state.
 