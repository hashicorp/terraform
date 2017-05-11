---
layout: "alicloud"
page_title: "Alicloud: alicloud_disk"
sidebar_current: "docs-alicloud-resource-disk"
description: |-
  Provides a ECS Disk resource.
---

# alicloud\_disk

Provides a ECS disk resource.

~> **NOTE:** One of `size` or `snapshot_id` is required when specifying an ECS disk. If all of them be specified, `size` must more than the size of snapshot which `snapshot_id` represents. Currently, `alicloud_disk` doesn't resize disk.

## Example Usage

```
# Create a new ECS disk.
resource "alicloud_disk" "ecs_disk" {
  # cn-beijing
  availability_zone = "cn-beijing-b"
  name              = "New-disk"
  description       = "Hello ecs disk."
  category          = "cloud_efficiency"
  size              = "30"

  tags {
    Name = "TerraformTest"
  }
}
```
## Argument Reference

The following arguments are supported:

* `availability_zone` - (Required, Forces new resource) The Zone to create the disk in.
* `name` - (Optional) Name of the ECS disk. This name can have a string of 2 to 128 characters, must contain only alphanumeric characters or hyphens, such as "-",".","_", and must not begin or end with a hyphen, and must not begin with http:// or https://. Default value is null.
* `description` - (Optional) Description of the disk. This description can have a string of 2 to 256 characters, It cannot begin with http:// or https://. Default value is null.
* `category` - (Optional, Forces new resource) Category of the disk. Valid values are `cloud`, `cloud_efficiency` and `cloud_ssd`. Default is `cloud`.
* `size` - (Required) The size of the disk in GiBs, and its value depends on `Category`. `cloud` disk value range: 5GB ~ 2000GB and other category disk value range: 20 ~ 32768.
* `snapshot_id` - (Optional) A snapshot to base the disk off of. If it is specified, `size` will be invalid and the disk size is equals to the snapshot size.
* `tags` - (Optional) A mapping of tags to assign to the resource.

## Attributes Reference

The following attributes are exported:

* `id` - The disk ID.
* `availability_zone` - The Zone to create the disk in.
* `name` - The disk name.
* `description` - The disk description.
* `status` - The disk status.
* `category` - The disk category.
* `size` - The disk size.
* `snapshot_id` - The disk snapshot ID.
* `tags` - The disk tags.