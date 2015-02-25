---
layout: "aws"
page_title: "AWS: aws_launch_configuration"
sidebar_current: "docs-aws-resource-launch-config"
description: |-
  Provides a resource to create a new launch configuration, used for autoscaling groups.
---

# aws\_launch\_configuration

Provides a resource to create a new launch configuration, used for autoscaling groups.

## Example Usage

```
resource "aws_launch_configuration" "as_conf" {
    name = "web_config"
    image_id = "ami-1234"
    instance_type = "m1.small"
}
```

## Argument Reference

The following arguments are supported:

* `name` - (Required) The name of the launch configuration.
* `image_id` - (Required) The EC2 image ID to launch.
* `instance_type` - (Required) The size of instance to launch.
* `iam_instance_profile` - (Optional) The IAM instance profile to associate
     with launched instances.
* `key_name` - (Optional) The key name that should be used for the instance.
* `security_groups` - (Optional) A list of associated security group IDS.
* `associate_public_ip_address` - (Optional) Associate a public ip address with an instance in a VPC.
* `user_data` - (Optional) The user data to provide when launching the instance.
* `block_device_mapping` - (Optional) A list of block devices to add. Their keys are documented below.

Each `block_device_mapping` supports the following:

* `device_name` - The name of the device to mount.
* `virtual_name` - (Optional) The virtual device name.
* `snapshot_id` - (Optional) The Snapshot ID to mount.
* `volume_type` - (Optional) The type of volume. Can be standard, gp2, or io1. Defaults to standard.
* `volume_size` - (Optional) The size of the volume in gigabytes.
* `delete_on_termination` - (Optional) Should the volume be destroyed on instance termination (defaults true).
* `encrypted` - (Optional) Should encryption be enabled (defaults false).

## Attributes Reference

The following attributes are exported:

* `id` - The ID of the launch configuration.
