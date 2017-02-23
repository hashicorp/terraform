---
layout: "backend-types"
page_title: "Backend Type: atlas"
sidebar_current: "docs-backends-types-standard-atlas"
description: |-
  Terraform can store the state in Atlas.
---

# atlas

**Kind: Standard (with no locking)**

Stores the state in [Atlas](https://atlas.hashicorp.com/).

You can create a new environment in the
[Environments section](https://atlas.hashicorp.com/environments)
and generate new token in the
[Tokens page](https://atlas.hashicorp.com/settings/tokens) under Settings.

## Example Configuration

```
terraform {
  backend "atlas" {
    name         = "bigbang/example"
    access_token = "foo"
  }
}
```

Note that for the access token we recommend using a
[partial configuration](/docs/backends/config.html).

## Example Referencing

```
data "terraform_remote_state" "foo" {
	backend = "atlas"
	config {
		name = "bigbang/example"
		access_token = "X2iTFefU5aWOjg.atlasv1.YaDa"
	}
}
```

## Configuration variables

The following configuration options / environment variables are supported:

 * `name` - (Required) Full name of the environment (`<username>/<name>`)
 * `access_token` / `ATLAS_TOKEN` - (Required) Atlas API token
 * `address` - (Optional) Address to alternative Atlas location (Atlas Enterprise endpoint)
