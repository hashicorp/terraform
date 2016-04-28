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
    image_id = "ami-408c7f28"
    instance_type = "t1.micro"
}
```

## Using with AutoScaling Groups

Launch Configurations cannot be updated after creation with the Amazon
Web Service API. In order to update a Launch Configuration, Terraform will
destroy the existing resource and create a replacement. In order to effectively
use a Launch Configuration resource with an [AutoScaling Group resource][1],
it's recommended to specify `create_before_destroy` in a [lifecycle][2] block.
Either omit the Launch Configuration `name` attribute, or specify a partial name
with `name_prefix`.  Example:

```
resource "aws_launch_configuration" "as_conf" {
    name_prefix = "terraform-lc-example-"
    image_id = "ami-408c7f28"
    instance_type = "t1.micro"

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

## Using with Spot Instances

Launch configurations can set the spot instance pricing to be used for the
Auto Scaling Group to reserve instances. Simply specifying the `spot_price`
parameter will set the price on the Launch Configuration which will attempt to
reserve your instances at this price.  See the [AWS Spot Instance
documentation](https://docs.aws.amazon.com/AWSEC2/latest/UserGuide/using-spot-instances.html)
for more information or how to launch [Spot Instances][3] with Terraform.

```
resource "aws_launch_configuration" "as_conf" {
    image_id = "ami-408c7f28"
    instance_type = "t1.micro"
    spot_price = "0.001"
    lifecycle {
      create_before_destroy = true
    }
}

resource "aws_autoscaling_group" "bar" {
    name = "terraform-asg-example"
    launch_configuration = "${aws_launch_configuration.as_conf.name}"
}
```

## Argument Reference

The following arguments are supported:

* `name` - (Optional) The name of the launch configuration. If you leave
  this blank, Terraform will auto-generate a unique name.
* `name_prefix` - (Optional) Creates a unique name beginning with the specified
  prefix. Conflicts with `name`.
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
* `spot_price` - (Optional) The price to use for reserving spot instances.
* `placement_tenancy` - (Optional) The tenancy of the instance. Valid values are
  `"default"` or `"dedicated"`, see [AWS's Create Launch Configuration](http://docs.aws.amazon.com/AutoScaling/latest/APIReference/API_CreateLaunchConfiguration.html)
  for more details

## Block devices

Each of the `*_block_device` attributes controls a portion of the AWS
Launch Configuration's "Block Device Mapping". It's a good idea to familiarize yourself with [AWS's Block Device
Mapping docs](https://docs.aws.amazon.com/AWSEC2/latest/UserGuide/block-device-mapping-concepts.html)
to understand the implications of using these attributes.

The `root_block_device` mapping supports the following:

* `volume_type` - (Optional) The type of volume. Can be `"standard"`, `"gp2"`,
  or `"io1"`. (Default: `"standard"`).
* `volume_size` - (Optional) The size of the volume in gigabytes.
* `iops` - (Optional) The amount of provisioned
  [IOPS](https://docs.aws.amazon.com/AWSEC2/latest/UserGuide/ebs-io-characteristics.html).
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
  [IOPS](https://docs.aws.amazon.com/AWSEC2/latest/UserGuide/ebs-io-characteristics.html).
  This must be set with a `volume_type` of `"io1"`.
* `delete_on_termination` - (Optional) Whether the volume should be destroyed
  on instance termination (Default: `true`).
* `encrypted` - (Optional) Whether the volume should be encrypted or not. Do not use this option if you are using `snapshot_id` as the encrypted flag will be determined by the snapshot. (Default: `false`).

Modifying any `ebs_block_device` currently requires resource replacement.

Each `ephemeral_block_device` supports the following:

* `device_name` - The name of the block device to mount on the instance.
* `virtual_name` - The [Instance Store Device
  Name](https://docs.aws.amazon.com/AWSEC2/latest/UserGuide/InstanceStorage.html#InstanceStoreDeviceNames)
  (e.g. `"ephemeral0"`)

Each AWS Instance type has a different set of Instance Store block devices
available for attachment. AWS [publishes a
list](https://docs.aws.amazon.com/AWSEC2/latest/UserGuide/InstanceStorage.html#StorageOnInstanceTypes)
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
[3]: /docs/providers/aws/r/spot_instance_request.html
