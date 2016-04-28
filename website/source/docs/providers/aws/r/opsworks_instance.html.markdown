---
layout: "aws"
page_title: "AWS: aws_opsworks_instance"
sidebar_current: "docs-aws-resource-opsworks-instance"
description: |-
  Provides an OpsWorks instance resource.
---

# aws\_opsworks\_instance

Provides an OpsWorks instance resource.

## Example Usage

```
resource "aws_opsworks_instance" "my-instance" {
  stack_id = "${aws_opsworks_stack.my-stack.id}"

  layer_ids = [
    "${aws_opsworks_custom_layer.my-layer.id}",
  ]

  instance_type = "t2.micro"
  os            = "Amazon Linux 2015.09"
  state         = "stopped"
}
```

## Argument Reference

The following arguments are supported:

* `instance_type` - (Required) The type of instance to start
* `stack_id` - (Required) The id of the stack the instance will belong to.
* `layer_ids` - (Required) The ids of the layers the instance will belong to.
* `state` - (Optional) The desired state of the instance.  Can be either `"running"` or `"stopped"`.
* `install_updates_on_boot` - (Optional) Controls where to install OS and package updates when the instance boots.  Defaults to `true`.
* `auto_scaling_type` - (Optional) Creates load-based or time-based instances.  If set, can be either: `"load"` or `"timer"`.
* `availability_zone` - (Optional) Name of the availability zone where instances will be created
  by default. 
* `ebs_optimized` - (Optional) If true, the launched EC2 instance will be EBS-optimized.
* `hostname` - (Optional) The instance's host name.
* `architecture` - (Optional) Machine architecture for created instances.  Can be either `"x86_64"` (the default) or `"i386"`
* `ami_id` - (Optional) The AMI to use for the instance.  If an AMI is specified, `os` must be `"Custom"`.
* `os` - (Optional) Name of operating system that will be installed.
* `root_device_type` - (Optional) Name of the type of root device instances will have by default.  Can be either `"ebs"` or `"instance-store"`
* `ssh_key_name` - (Optional) Name of the SSH keypair that instances will have by default.
* `agent_version` - (Optional) The AWS OpsWorks agent to install.  Defaults to `"INHERIT"`.
* `subnet_id` - (Optional) Subnet ID to attach to
* `virtualization_type` - (Optional) Keyword to choose what virtualization mode created instances
  will use. Can be either `"paravirtual"` or `"hvm"`.
* `root_block_device` - (Optional) Customize details about the root block
  device of the instance. See [Block Devices](#block-devices) below for details.
* `ebs_block_device` - (Optional) Additional EBS block devices to attach to the
  instance.  See [Block Devices](#block-devices) below for details.
* `ephemeral_block_device` - (Optional) Customize Ephemeral (also known as
  "Instance Store") volumes on the instance. See [Block Devices](#block-devices) below for details.


## Block devices

Each of the `*_block_device` attributes controls a portion of the AWS
Instance's "Block Device Mapping". It's a good idea to familiarize yourself with [AWS's Block Device
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

* `device_name` - The name of the device to mount.
* `snapshot_id` - (Optional) The Snapshot ID to mount.
* `volume_type` - (Optional) The type of volume. Can be `"standard"`, `"gp2"`,
  or `"io1"`. (Default: `"standard"`).
* `volume_size` - (Optional) The size of the volume in gigabytes.
* `iops` - (Optional) The amount of provisioned
  [IOPS](http://docs.aws.amazon.com/AWSEC2/latest/UserGuide/ebs-io-characteristics.html).
  This must be set with a `volume_type` of `"io1"`.
* `delete_on_termination` - (Optional) Whether the volume should be destroyed
  on instance termination (Default: `true`).
* `encrypted` - (Optional) Enables [EBS
  encryption](http://docs.aws.amazon.com/AWSEC2/latest/UserGuide/EBSEncryption.html)
  on the volume (Default: `false`). Cannot be used with `snapshot_id`.

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

~> **NOTE:** Currently, changes to `*_block_device` configuration of _existing_
resources cannot be automatically detected by Terraform. After making updates
to block device configuration, resource recreation can be manually triggered by
using the [`taint` command](/docs/commands/taint.html).


## Attributes Reference

The following attributes are exported:

* `id` - The id of the OpsWorks instance.
* `agent_version` - The AWS OpsWorks agent version.
* `availability_zone` - The availability zone of the instance.
* `ssh_key_name` - The key name of the instance
* `public_dns` - The public DNS name assigned to the instance. For EC2-VPC, this 
  is only available if you've enabled DNS hostnames for your VPC
* `public_ip` - The public IP address assigned to the instance, if applicable.
* `private_dns` - The private DNS name assigned to the instance. Can only be 
  used inside the Amazon EC2, and only available if you've enabled DNS hostnames 
  for your VPC
* `private_ip` - The private IP address assigned to the instance
* `subnet_id` - The VPC subnet ID.
* `security_group_ids` - The associated security groups.

