---
layout: "remotestate"
page_title: "Remote State Backend: http"
sidebar_current: "docs-state-remote-http"
description: |-
  Terraform can store the state remotely, making it easier to version and work with in a team.
---

# http

Stores the state using a simple [REST](https://en.wikipedia.org/wiki/Representational_state_transfer) client.

State will be fetched via GET, updated via POST, and purged with DELETE.

## Example Usage

```
terraform remote config \
	-backend=http \
	-backend-config="address=http://my.rest.api.com"
```

## Example Referencing

```
resource "terraform_remote_state" "foo" {
	backend = "http"
	config {
		address = "http://my.rest.api.com"
	}
}
```

## Configuration variables

The following configuration options are supported:

 * `address` - (Required) The address of the REST endpoint
 * `skip_cert_verification` - (Optional) Whether to skip TLS verification.
   Defaults to `false`.
