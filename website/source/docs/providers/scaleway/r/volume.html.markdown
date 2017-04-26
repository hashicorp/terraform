---
layout: "scaleway"
page_title: "Scaleway: volume"
sidebar_current: "docs-scaleway-resource-volume"
description: |-
  Manages Scaleway Volumes.
---

# scaleway\_volume

Provides volumes. This allows volumes to be created, updated and deleted.
For additional details please refer to [API documentation](https://developer.scaleway.com/#volumes).

## Example Usage

```hcl
resource "scaleway_server" "test" {
  name    = "test"
  image   = "aecaed73-51a5-4439-a127-6d8229847145"
  type    = "C2S"
  volumes = ["${scaleway_volume.test.id}"]
}

resource "scaleway_volume" "test" {
  name       = "test"
  size_in_gb = 20
  type       = "l_ssd"
}
```

## Argument Reference

The following arguments are supported:

* `name` - (Required) name of volume
* `size_in_gb` - (Required) size of the volume in GB
* `type` - (Required) type of volume

## Attributes Reference

The following attributes are exported:

* `id` - id of the new resource

## Import

Instances can be imported using the `id`, e.g.

```
$ terraform import scaleway_volume.test 5faef9cd-ea9b-4a63-9171-9e26bec03dbc
```
