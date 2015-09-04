---
layout: "aws"
page_title: "AWS: aws_ami"
sidebar_current: "docs-aws-resource-ami"
description: |-
  Creates and manages a custom Amazon Machine Image (AMI).
---

# aws\_ami

The AMI resource allows the creation and management of a completely-custom
*Amazon Machine Image* (AMI).

If you just want to duplicate an existing AMI, possibly copying it to another
region, it's better to use `aws_ami_copy` instead.

## Example Usage

```
# Create an AMI that will start a machine whose root device is backed by
# an EBS volume populated from a snapshot. It is assumed that such a snapshot
# already exists with the id "snap-xxxxxxxx".
resource "aws_ami" "example" {
    name = "terraform-example"
    virtualization_type = "hvm"
    root_device_name = "/dev/xvda"

    ebs_block_device {
        device_name = "/dev/xvda"
        snapshot_id = "snap-xxxxxxxx"
        volume_size = 8
    }
}
```

## Argument Reference

The following arguments are supported:

* `name` - (Required) A region-unique name for the AMI.
* `description` - (Optional) A longer, human-readable description for the AMI.
* `virtualization_type` - (Optional) Keyword to choose what virtualization mode created instances
  will use. Can be either "paravirtual" (the default) or "hvm". The choice of virtualization type
  changes the set of further arguments that are required, as described below.
* `architecture` - (Optional) Machine architecture for created instances. Defaults to "x86_64".
* `ebs_block_device` - (Optional) Nested block describing an EBS block device that should be
  attached to created instances. The structure of this block is described below.
* `ephemeral_block_device` - (Optional) Nested block describing an ephemeral block device that
  should be attached to created instances. The structure of this block is described below.

When `virtualization_type` is "paravirtual" the following additional arguments apply:

* `image_location` - (Required) Path to an S3 object containing an image manifest, e.g. created
  by the `ec2-upload-bundle` command in the EC2 command line tools.
* `kernel_id` - (Required) The id of the kernel image (AKI) that will be used as the paravirtual
  kernel in created instances.
* `ramdisk_id` - (Optional) The id of an initrd image (ARI) that will be used when booting the
  created instances.

When `virtualization_type` is "hvm" the following additional arguments apply:

* `sriov_net_support` - (Optional) When set to "simple" (the default), enables enhanced networking
  for created instances. No other value is supported at this time.

Nested `ebs_block_device` blocks have the following structure:

* `device_name` - (Required) The path at which the device is exposed to created instances.
* `delete_on_termination` - (Optional) Boolean controlling whether the EBS volumes created to
  support each created instance will be deleted once that instance is terminated.
* `encrypted` - (Optional) Boolean controlling whether the created EBS volumes will be encrypted.
* `iops` - (Required only when `volume_type` is "io1") Number of I/O operations per second the
  created volumes will support.
* `snapshot_id` - (Optional) The id of an EBS snapshot that will be used to initialize the created
  EBS volumes. If set, the `volume_size` attribute must be at least as large as the referenced
  snapshot.
* `volume_size` - (Required unless `snapshot_id` is set) The size of created volumes in GiB.
  If `snapshot_id` is set and `volume_size` is omitted then the volume will have the same size
  as the selected snapshot.
* `volume_type` - (Optional) The type of EBS volume to create. Can be one of "standard" (the
  default), "io1" or "gp2".

Nested `ephemeral_block_device` blocks have the following structure:

* `device_name` - (Required) The path at which the device is exposed to created instances.
* `virtual_name` - (Required) A name for the ephemeral device, of the form "ephemeralN" where
  *N* is a volume number starting from zero.

## Attributes Reference

The following attributes are exported:

* `id` - The ID of the created AMI.
