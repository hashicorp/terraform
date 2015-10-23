---
layout: "google"
page_title: "Google: google_compute_disk"
sidebar_current: "docs-google-compute-disk"
description: |-
  Creates a new persistent disk within GCE, based on another disk.
---

# google\_compute\_disk

Creates a new persistent disk within GCE, based on another disk.

## Example Usage

```
resource "google_compute_disk" "default" {
	name = "test-disk"
	type = "pd-ssd"
	zone = "us-central1-a"
	image = "debian7-wheezy"
}
```

## Argument Reference

The following arguments are supported:

* `name` - (Required) A unique name for the resource, required by GCE.
    Changing this forces a new resource to be created.

* `zone` - (Required) The zone where this disk will be available.

* `image` - (Optional) The image from which to initialize this disk.  Either the full URL, a
  contraction of the form "project/name", or just a name (in which case the current project is
used).

* `snapshot` - (Optional) Name of snapshot from which to initialize this disk;

* `size` - (Optional) The size of the image in gigabytes. If not specified,
    it will inherit the size of its base image.

* `type` - (Optional) The GCE disk type.

## Attributes Reference

The following attributes are exported:

* `name` - The name of the resource.
* `zone` - The zone where the resource is located.
* `image` - The name of the image the disk is based off of.
* `size` - The size of the disk in gigabytes.
