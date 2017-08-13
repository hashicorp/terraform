---
layout: "backend-types"
page_title: "Backend Type: http"
sidebar_current: "docs-backends-types-standard-http"
description: |-
  Terraform can store state remotely at any valid HTTP endpoint.
---

# http

**Kind: Standard (with optional locking)**

Stores the state using a simple [REST](https://en.wikipedia.org/wiki/Representational_state_transfer) client.

State will be fetched via GET, updated via POST, and purged with DELETE.

If locking is enabled a lock POST request will be made by appending `/lock` to the configured address with the lock info in the
body as json. The endpiont should return a 409 Conflict with the holding lock info when it's already taken, 200 OK for success.
Any other status will be considered an error. Unlocking works simillarly, appending `/unlock` and adding the `ID` of the lock
being freed as a query parameter. An `ID` query parameter will also be added when sending state updates.

## Example Usage

```hcl
terraform {
  backend "http" {
    address = "http://myrest.api.com"
  }
}
```

## Example Referencing

```hcl
data "terraform_remote_state" "foo" {
  backend = "http"
  config {
    address = "http://my.rest.api.com"
  }
}
```

## Configuration variables

The following configuration options are supported:

 * `address` - (Required) The address of the REST endpoint
 * `username` - (Optional) The username for HTTP basic authentication
 * `password` - (Optional) The password for HTTP basic authentication
 * `skip_cert_verification` - (Optional) Whether to skip TLS verification.
   Defaults to `false`.
 * `supports_locking` - (Optional) Whether to enable locking related calls
   Defaults to `false`.
