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

```
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

* `datacenter` - (Optional)[string] Location of desired 1and1 datacenter ["DE", "GB", "US", "ES" ]
* `description` - (Optional)[string] Description for the shared storage
* `size` - (Required)[string] Size of the shared storage
* `storage_servers`  (Optional)[Collection] List of servers that will have access to the stored storage
    * `id` - (Required) [string] ID of the server
    * `rights` - (Required)[string] Access rights to be assigned to the server ["RW","R"]
