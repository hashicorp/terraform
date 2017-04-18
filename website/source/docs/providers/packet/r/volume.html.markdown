---
layout: "packet"
page_title: "Packet: packet_volume"
sidebar_current: "docs-packet-resource-volume"
description: |-
  Provides a Packet Block Storage Volume Resource.
---

# packet\_volume

Provides a Packet Block Storage Volume resource to allow you to
manage block volumes on your account.
Once created by Terraform, they must then be attached and mounted
using the api and `packet_block_attach` and `packet_block_detach`
scripts.

## Example Usage

```hcl
# Create a new block volume
resource "packet_volume" "volume1" {
  description   = "terraform-volume-1"
  facility      = "ewr1"
  project_id    = "${packet_project.cool_project.id}"
  plan          = "storage_1"
  size          = 100
  billing_cycle = "hourly"

  snapshot_policies = {
    snapshot_frequency = "1day"

    snapshot_count = 7
  }

  snapshot_policies = {
    snapshot_frequency = "1month"

    snapshot_count = 6
  }
}
```

## Argument Reference

The following arguments are supported:

* `plan` - (Required) The service plan slug of the volume
* `facility` - (Required) The facility to create the volume in
* `project_id` - (Required) The packet project ID to deploy the volume in
* `size` - (Required) The size in GB to make the volume
* `billing_cycle` - The billing cycle, defaults to "hourly"
* `description` - Optional description for the volume
* `snapshot_policies` - Optional list of snapshot policies

## Attributes Reference

The following attributes are exported:

* `id` - The unique ID of the volume
* `name` - The name of the volume
* `description` - The description of the volume
* `size` - The size in GB of the volume
* `plan` - Performance plan the volume is on
* `billing_cycle` - The billing cycle, defaults to hourly
* `facility` - The facility slug the volume resides in
* `state` - The state of the volume
* `locked` - Whether the volume is locked or not
* `project_id ` - The project id the volume is in
* `created` - The timestamp for when the volume was created
* `updated` - The timestamp for the last time the volume was updated
