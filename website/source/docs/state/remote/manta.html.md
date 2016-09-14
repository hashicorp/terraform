---
layout: "remotestate"
page_title: "Remote State Backend: manta"
sidebar_current: "docs-state-remote-manta"
description: |-
  Terraform can store the state remotely, making it easier to version and work with in a team.
---

# manta

Stores the state as an artifact in [Manta](https://www.joyent.com/manta).

## Example Usage

```
terraform remote config \
	-backend=manta \
	-backend-config="path=random/path" \
  -backend-config="objecName=tfstate.tf"
```

## Example Referencing

```
data "terraform_remote_state" "foo" {
	backend = "manta"
	config {
		path = "random/path"
    objectName = "tfstate.tf"
	}
}
```

## Configuration variables

The following configuration option is supported:

 * `path` - (Required) The path where to store the state file
 * `objectName` - (Optional) The name of the state file (defaults to `tfstate.tf`)
 * `keyName` - (Optional) The path to your private key for accessing Manta (defaults to `~/.ssh/id_rsa`)

The following [Manta environment variables](https://apidocs.joyent.com/manta/#setting-up-your-environment) are supported:

 * `MANTA_USER` - (Required) The Manta user
 * `MANTA_KEY_ID` - (Required) The fingerprint of your SSH key
 * `MANTA_URL` - (Required) The API endpoint
