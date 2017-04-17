---
layout: "digitalocean"
page_title: "DigitalOcean: digitalocean_volume"
sidebar_current: "docs-do-resource-volume"
description: |-
  Provides a DigitalOcean volume resource.
---

# digitalocean\_volume

Provides a DigitalOcean Block Storage volume which can be attached to a Droplet in order to provide expanded storage.

## Example Usage

```hcl
resource "digitalocean_volume" "foobar" {
  region      = "nyc1"
  name        = "baz"
  size        = 100
  description = "an example volume"
}

resource "digitalocean_droplet" "foobar" {
  name       = "baz"
  size       = "1gb"
  image      = "coreos-stable"
  region     = "nyc1"
  volume_ids = ["${digitalocean_volume.foobar.id}"]
}
```

## Argument Reference

The following arguments are supported:

* `region` - (Required) The region that the block storage volume will be created in.
* `name` - (Required) A name for the block storage volume. Must be lowercase and be composed only of numbers, letters and "-", up to a limit of 64 characters.
* `size` - (Required) The size of the block storage volume in GiB.
* `description` - (Optional) A free-form text field up to a limit of 1024 bytes to describe a block storage volume.

## Attributes Reference

The following attributes are exported:

* `id` - The unique identifier for the block storage volume.


## Import

Volumes can be imported using the `volume id`, e.g.

```
terraform import digitalocean_volume.volumea 506f78a4-e098-11e5-ad9f-000f53306ae1
```