---
layout: "backend-types"
page_title: "Backend Type: MySQL"
sidebar_current: "docs-backends-types-standard-mysql"
description: |-
  Terraform can store state remotely in a MySQL database with locking.
---

# MySQL

**Kind: Standard (with locking)**

Stores the state in a [MySQL](https://www.mysql.com) or [MariaDB](https://www.mariadb.org) database.

This backend supports [state locking](/docs/state/locking.html).

## Example Configuration

```hcl
terraform {
  backend "mysql" {
    conn_str = "user:pass@mysqldb.example.com/terraform_backend"
  }
}
```

Before initializing the backend with `terraform init`, the database must already exist:

```
mysql> CREATE DATABASE terraform_backend
```

This `CREATE DATABASE` command is found in [CREATE DATABASE Statement](https://dev.mysql.com/doc/refman/8.0/en/create-database.html) which are installed along with the database server.

We recommend using a [partial configuration](/docs/backends/config.html#partial-configuration) for the `conn_str` variable, because it typically contains access credentials that should not be committed to source control:

```hcl
terraform {
  backend "mysql" {}
}
```

Then, set the credentials when initializing the configuration:

```
terraform init -backend-config="conn_str=user:pass@mysqldb.example.com/terraform_backend"
```

To use a Mysql server running on the same machine as Terraform, configure localhost with SSL disabled:

```
terraform init -backend-config="conn_str=user:pass@mysqldb.example.com/terraform_backend?sslMode=DISABLED"
```

## Example Referencing

To make use of the mysql remote state we can use the [`terraform_remote_state` data source](/docs/providers/terraform/d/remote_state.html).

```hcl
data "terraform_remote_state" "network" {
  backend = "mysql"
  config {
    conn_str = "user:pass@mysqldb.example.com/terraform_backend"
  }
}
```

## Configuration Variables

The following configuration options or environment variables are supported:

 * `conn_str` - (Required) Mysql connection string; a `user:pass@mysqldb.example.com/` URL
 * `schema_name` - Name of the automatically-managed Mysql schema, default `terraform_remote_state`.
 * `skip_schema_creation` - If set to `true`, the Mysql schema must already exist. Terraform won't try to create the schema. Useful when the Mysql user does not have "create schema" permission on the database.

## Technical Design

Mysql version 5.7 or newer is required to support advisory locks.

This backend creates one table **states** in the automatically-managed Mysql schema configured by the `schema_name` variable.

The table is keyed by the [workspace](/docs/state/workspaces.html) name. If workspaces are not in use, the name `default` is used.

Locking is supported using [Mysql Locking Functions](https://dev.mysql.com/doc/refman/8.0/en/locking-functions.html). To see outstanding locks in a Mysql server, see [Metadata Locking](https://dev.mysql.com/doc/refman/8.0/en/metadata-locking.html).

The **states** table contains:

 * a serial integer `id`, used as the key for advisory locks
 * the workspace `name` key as *text* with a unique index
 * the Terraform state `data` as *text*
