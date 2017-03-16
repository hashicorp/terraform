---
layout: "rancher"
page_title: "Rancher: rancher_host"
sidebar_current: "docs-rancher-resource-host"
description: |-
  Provides a Rancher Host resource. This can be used to manage and delete hosts on Rancher.
---

# rancher\_host

Provides a Rancher Host resource. This can be used to manage and delete hosts on Rancher.

## Example usage

```hcl
# Manage an existing Rancher host
resource rancher_host "foo" {
  name           = "foo"
  description    = "The foo node"
  environment_id = "1a5"
  hostname       = "foo.example.com"
  labels {
    role = "database"
  }
}
```

## Argument Reference

The following arguments are supported:

* `name` - (Required) The name of the host.
* `description` - (Optional) A host description.
* `environment_id` - (Required) The ID of the environment the host is associated to.
* `hostname` - (Required) The host name. Used as the primary key to detect the host ID.
* `labels` - (Optional) A dictionary of labels to apply to the host. Computed internal labels are excluded from that list.
