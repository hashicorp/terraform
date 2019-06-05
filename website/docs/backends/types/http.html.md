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

State will be fetched via GET, updated via POST, and purged with DELETE. The method used for updating is configurable.

When locking support is enabled it will use LOCK and UNLOCK requests providing the lock info in the body. The endpoint should
return a 423: Locked or 409: Conflict with the holding lock info when it's already taken, 200: OK for success. Any other status
will be considered an error. The ID of the holding lock info will be added as a query parameter to state updates requests.

## Example Usage

```hcl
terraform {
  backend "http" {
    address = "http://myrest.api.com/foo"
    lock_address = "http://myrest.api.com/foo"
    unlock_address = "http://myrest.api.com/foo"
  }
}
```

## Example Referencing

```hcl
data "terraform_remote_state" "foo" {
  backend = "http"
  config = {
    address = "http://my.rest.api.com"
  }
}
```

## Configuration variables

The following configuration options are supported:

 * `address` - (Required) The address of the REST endpoint
 * `update_method` - (Optional) HTTP method to use when updating state.
   Defaults to `POST`.
 * `lock_address` - (Optional) The address of the lock REST endpoint.
   Defaults to disabled.
 * `lock_method` - (Optional) The HTTP method to use when locking.
   Defaults to `LOCK`.
 * `unlock_address` - (Optional) The address of the unlock REST endpoint.
   Defaults to disabled.
 * `unlock_method` - (Optional) The HTTP method to use when unlocking.
   Defaults to `UNLOCK`.
 * `username` - (Optional) The username for HTTP basic authentication
 * `password` - (Optional) The password for HTTP basic authentication
 * `skip_cert_verification` - (Optional) Whether to skip TLS verification.
   Defaults to `false`.
 * `retry_max` – (Optional) The number of HTTP request retries. Defaults to `2`.
 * `retry_wait_min` – (Optional) The minimum time in seconds to wait between HTTP request attempts.
   Defaults to `1`.
 * `retry_wait_max` – (Optional) The maximum time in seconds to wait between HTTP request attempts.
   Defaults to `30`.
