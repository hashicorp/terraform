---
layout: "oneandone"
page_title: "1&1: oneandone_shared_storage"
sidebar_current: "docs-oneandone-resource-shared-storage"
description: |-
  Creates and manages 1&1 Shared Storage.
---

# oneandone\_server

Manages a Shared Storage on 1&1

## Example Usage

```hcl
resource "oneandone_shared_storage" "storage" {
  name = "test_storage1"
  description = "1234"
  size = 50

  storage_servers = [
    {
      id = "${oneandone_server.server.id}"
      rights = "RW"
    },
    {
      id = "${oneandone_server.server02.id}"
      rights = "RW"
    }
  ]
}
```

## Argument Reference

The following arguments are supported:

* `datacenter` - (Optional) Location of desired 1and1 datacenter. Can be `DE`, `GB`, `US` or `ES`
* `description` - (Optional) Description for the shared storage
* `size` - (Required) Size of the shared storage
* `storage_servers`  (Optional) List of servers that will have access to the stored storage
    * `id` - (Required) ID of the server
    * `rights` - (Required) Access rights to be assigned to the server. Can be `RW` or `R`
