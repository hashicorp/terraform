---
layout: "digitalocean"
page_title: "DigitalOcean: digitalocean_image"
sidebar_current: "docs-do-datasource-image"
description: |-
  Get information on an snapshot.
---

# digitalocean_image

Get information on an snapshot images. The aim of this datasource is to enable
you to build droplets based on snapshot names.

An error is triggered if zero or more than one result is returned by the query.

## Example Usage

Get the data about a snapshot:

```hcl
data "digitalocean_image" "example1" {
  name = "example-1.0.0"
}
```

Reuse the data about a snapshot to create a droplet:

```hcl
data "digitalocean_image" "example1" {
  name = "example-1.0.0"
}
resource "digitalocean_droplet" "example1" {
  image  = "${data.digitalocean_image.example1.image}"
  name   = "example-1"
  region = "nyc2"
  size   = "512mb"
}
```

## Argument Reference

The following arguments are supported:

* `name` - The name of the image.

## Attributes Reference

The following attributes are exported:

* `name` - See Argument Reference above.
* `image` - The id of the image.
* `min_disk_size`: The minimum 'disk' required for the image.
* `private` - Is image a public image or not. Public images represents
  Linux distributions or Application, while non-public images represent
  snapshots and backups and are only available within your account.
* `regions`: The regions that the image is available in.
* `size_gigabytes`: The size of the image in gigabytes.
* `type`: Type of the image. Can be "snapshot" or "backup".
