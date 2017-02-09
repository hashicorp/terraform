---
layout: "swift"
page_title: "Swift: container"
sidebar_current: "docs-swift-resource-container"
description: |-
  Manages Swift Object Storage containers.
---

# swift_container

Creates an object container in a Swift object store.

## Example Usage

```
resource "swift_container" "test_container" {
    name = "test_containe_name"
    read_access = ["joe", "pete"]
    write_access = ["bob"]
}
```

## Argument Reference

The following arguments are supported:

* `name` - (Required) The name of the object container.
* `read_access` - (Optional) A list of usernames that will have read access to objects in this container.
* `write_access` - (Optional) A list of usernames that will have write access to objects in this container.

Fields `read_access` and `write_access` are editable.
