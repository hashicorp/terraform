---
layout: "digitalocean"
page_title: "DigitalOcean: digitalocean_image"
sidebar_current: "docs-do-datasource-image"
description: |-
  Get information on an image.
---

# digitalocean_image

Get information on an image. This can refer to public Linux distributions,
applications, private backups or snapshots.

An error is triggered if more than one result is returned by the query.

## Example Usage

Option 1: From a name:

Reference the name:

```hcl
data "digitalocean_snapshot" "example1" {
  name = "example-1.0.0"
}
```

Option 2: From a slug:

Reference the name:

```hcl
data "digitalocean_snapshot" "example1" {
  name = "centos-7-x86"
}
```

## Argument Reference

The following arguments are supported:

* `name` - (Optional) The name of the image.
* `slug` - (Optional) The slug of the image.
* `private` - (Optional) Look for public images or not. Public images represents
  Linux distributions or Application, while non-public images represent
  snapshots and backups and are only available within your account.

## Attributes Reference

The following attributes are exported:

* `name` - See Argument Reference above.
* `slug` - See Argument Reference above.
* `private` - See Argument Reference above.
* `id` - The id of the snapshot, can be used to create droplets from the
  snapshot.
* `regions`: The regions that the image is available in.
* `regions`: Type of the image. Can be "snapshot" or "backup".
* `min_disk_size`: The minimum 'disk' required for the image.
* `size_gigabytes`: The size of the image in gigabytes.

