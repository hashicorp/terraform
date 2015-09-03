---
layout: "aws"
page_title: "AWS: aws_launch_configuration"
sidebar_current: "docs-aws-resource-launch-configuration"
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

## Using with AutoScaling Groups

Launch Configurations cannot be updated after creation with the Amazon
Web Service API. In order to update a Launch Configuration, Terraform will
destroy the existing resource and create a replacement. If order to effectively
use a Launch Configuration resource with an[AutoScaling Group resource][1],
it's recommend to omit the Launch Configuration `name` attribute, and
specify `create_before_destroy` in a [lifecycle][2] block, as shown:

```
resource "aws_launch_configuration" "as_conf" {
    image_id = "ami-1234"
    instance_type = "m1.small"

    lifecycle {
      create_before_destroy = true
    }
}

resource "aws_autoscaling_group" "bar" {
    name = "terraform-asg-example"
    launch_configuration = "${aws_launch_configuration.as_conf.name}"

    lifecycle {
      create_before_destroy = true
    }
}
```

With this setup Terraform generates a unique name for your Launch
Configuration and can then update the AutoScaling Group without conflict before
destroying the previous Launch Configuration.

## Argument Reference

The following arguments are supported:

* `name` - (Optional) The name of the launch configuration. If you leave
  this blank, Terraform will auto-generate it.
* `image_id` - (Required) The EC2 image ID to launch.
* `instance_type` - (Required) The size of instance to launch.
* `iam_instance_profile` - (Optional) The IAM instance profile to associate
     with launched instances.
* `key_name` - (Optional) The key name that should be used for the instance.
* `security_groups` - (Optional) A list of associated security group IDS.
* `associate_public_ip_address` - (Optional) Associate a public ip address with an instance in a VPC.
* `user_data` - (Optional) The user data to provide when launching the instance.
* `enable_monitoring` - (Optional) Enables/disables detailed monitoring. This is enabled by default.
* `ebs_optimized` - (Optional) If true, the launched EC2 instance will be EBS-optimized.
* `root_block_device` - (Optional) Customize details about the root block
  device of the instance. See [Block Devices](#block-devices) below for details.
* `ebs_block_device` - (Optional) Additional EBS block devices to attach to the
  instance.  See [Block Devices](#block-devices) below for details.
* `ephemeral_block_device` - (Optional) Customize Ephemeral (also known as
  "Instance Store") volumes on the instance. See [Block Devices](#block-devices) below for details.

<a id="block-devices"></a>
## Block devices

Each of the `*_block_device` attributes controls a portion of the AWS
Launch Configuration's "Block Device Mapping". It's a good idea to familiarize yourself with [AWS's Block Device
Mapping docs](http://docs.aws.amazon.com/AWSEC2/latest/UserGuide/block-device-mapping-concepts.html)
to understand the implications of using these attributes.

The `root_block_device` mapping supports the following:

* `volume_type` - (Optional) The type of volume. Can be `"standard"`, `"gp2"`,
  or `"io1"`. (Default: `"standard"`).
* `volume_size` - (Optional) The size of the volume in gigabytes.
* `iops` - (Optional) The amount of provisioned
  [IOPS](http://docs.aws.amazon.com/AWSEC2/latest/UserGuide/ebs-io-characteristics.html).
  This must be set with a `volume_type` of `"io1"`.
* `delete_on_termination` - (Optional) Whether the volume should be destroyed
  on instance termination (Default: `true`).

Modifying any of the `root_block_device` settings requires resource
replacement.

Each `ebs_block_device` supports the following:

* `device_name` - (Required) The name of the device to mount.
* `snapshot_id` - (Optional) The Snapshot ID to mount.
* `volume_type` - (Optional) The type of volume. Can be `"standard"`, `"gp2"`,
  or `"io1"`. (Default: `"standard"`).
* `volume_size` - (Optional) The size of the volume in gigabytes.
* `iops` - (Optional) The amount of provisioned
  [IOPS](http://docs.aws.amazon.com/AWSEC2/latest/UserGuide/ebs-io-characteristics.html).
  This must be set with a `volume_type` of `"io1"`.
* `delete_on_termination` - (Optional) Whether the volume should be destroyed
  on instance termination (Default: `true`).

Modifying any `ebs_block_device` currently requires resource replacement.

Each `ephemeral_block_device` supports the following:

* `device_name` - The name of the block device to mount on the instance.
* `virtual_name` - The [Instance Store Device
  Name](http://docs.aws.amazon.com/AWSEC2/latest/UserGuide/InstanceStorage.html#InstanceStoreDeviceNames)
  (e.g. `"ephemeral0"`)

Each AWS Instance type has a different set of Instance Store block devices
available for attachment. AWS [publishes a
list](http://docs.aws.amazon.com/AWSEC2/latest/UserGuide/InstanceStorage.html#StorageOnInstanceTypes)
of which ephemeral devices are available on each type. The devices are always
identified by the `virtual_name` in the format `"ephemeral{0..N}"`.

~> **NOTE:** Changes to `*_block_device` configuration of _existing_ resources
cannot currently be detected by Terraform. After updating to block device
configuration, resource recreation can be manually triggered by using the
[`taint` command](/docs/commands/taint.html).

## Attributes Reference

The following attributes are exported:

* `id` - The ID of the launch configuration.

[1]: /docs/providers/aws/r/autoscaling_group.html
[2]: /docs/configuration/resources.html#lifecycle
