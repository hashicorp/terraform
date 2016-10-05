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
	-backend-config="objecName=terraform.tfstate"
```

## Example Referencing

```
data "terraform_remote_state" "foo" {
	backend = "manta"
	config {
		path = "random/path"
		objectName = "terraform.tfstate"
	}
}
```

## Configuration variables

The following configuration options are supported:

 * `path` - (Required) The path where to store the state file
 * `objectName` - (Optional) The name of the state file (defaults to `terraform.tfstate`)

The following [Manta environment variables](https://apidocs.joyent.com/manta/#setting-up-your-environment) are supported:

 * `MANTA_URL` - (Required) The API endpoint
 * `MANTA_USER` - (Required) The Manta user
 * `MANTA_KEY_ID` - (Required) The MD5 fingerprint of your SSH key
 * `MANTA_KEY_MATERIAL` - (Required) The path to the private key for accessing Manta (must align with the `MANTA_KEY_ID`). This key must *not* be protected by passphrase.
