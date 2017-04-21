---
layout: "alicloud"
page_title: "Alicloud: alicloud_images"
sidebar_current: "docs-alicloud-datasource-images"
description: |-
    Provides a list of images available to the user.
---

# alicloud_images

The Images data source list image resource list contains private images of the user and images of system resources provided by Alicloud, as well as other public images and those available on the image market.

## Example Usage

```hcl
data "alicloud_images" "multi_image" {
  owners     = "system"
  name_regex = "^centos_6"
}
```

## Argument Reference

The following arguments are supported:

* `name_regex` - (Optional) A regex string to apply to the image list returned by Alicloud.
* `most_recent` - (Optional) If more than one result is returned, use the most recent image.
* `owners` - (Optional) Limit search to specific image owners. Valid items are `system`, `self`, `others`, `marketplace`.

## Attributes Reference

The following attributes are exported:

* `id` - ID of the image.
* `architecture` - Platform type of the image system:i386 | x86_64.
* `creation_time` - Time of creation.
* `description` - Description of the image.
* `image_owner_alias` - Alias of the image owner.
* `os_name` - Display name of the OS.
* `status` - Status of the image, with possible values: `UnAvailable`, `Available`, `Creating` or `CreateFailed`.
* `size` - Size of the image.
* `disk_device_mappings` - Description of the system with disks and snapshots under an image.
  * `device` - Device information of the created disk: such as /dev/xvdb.
  * `size` - Size of the created disk.
  * `snapshot_id` - Snapshot ID.
* `product_code` - Product code of the image on the image market.
* `is_subscribed` - Whether the user has subscribed to the terms of service for the image product corresponding to the ProductCode.
* `image_version` - Version of the image.
* `progress` - Progress of image creation, presented in percentages.
