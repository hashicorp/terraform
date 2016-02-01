---
layout: "remotestate"
page_title: "Remote State Backend: atlas"
sidebar_current: "docs-state-remote-atlas"
description: |-
  Terraform can store the state remotely, making it easier to version and work with in a team.
---

# atlas

Stores the state in [Atlas](https://atlas.hashicorp.com/).

You can create a new environment in the [Environments section](https://atlas.hashicorp.com/environments)
and generate new token in the [Tokens page](https://atlas.hashicorp.com/settings/tokens) under Settings.

## Example Usage

```
terraform remote config \
	-backend=atlas \
	-backend-config="name=bigbang/example" \
	-backend-config="access_token=X2iTFefU5aWOjg.atlasv1.YaDa" \
```

## Example Referencing

```
resource "terraform_remote_state" "foo" {
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
