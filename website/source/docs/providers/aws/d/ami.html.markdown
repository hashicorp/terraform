---
layout: "aws"
page_title: "AWS: aws_ami"
sidebar_current: "docs-aws-datasource-ami"
description: |-
  Get information on a Amazon Machine Image (AMI).
---

# aws\_ami

Use this data source to get the ID of a registered AMI for use in other
resources.

## Example Usage

```hcl
data "aws_ami" "nat_ami" {
  most_recent      = true
  executable_users = ["self"]

  filter {
    name   = "owner-alias"
    values = ["amazon"]
  }

  filter {
    name   = "name"
    values = ["amzn-ami-vpc-nat*"]
  }

  name_regex = "^myami-\\d{3}"
  owners     = ["self"]
}
```

## Argument Reference

* `most_recent` - (Optional) If more than one result is returned, use the most
recent AMI.

* `executable_users` - (Optional) Limit search to users with *explicit* launch permission on
 the image. Valid items are the numeric account ID or `self`.

* `filter` - (Optional) One or more name/value pairs to filter off of. There are
several valid keys, for a full reference, check out
[describe-images in the AWS CLI reference][1].

* `owners` - (Optional) Limit search to specific AMI owners. Valid items are the numeric
account ID, `amazon`, or `self`.

* `name_regex` - (Optional) A regex string to apply to the AMI list returned
by AWS. This allows more advanced filtering not supported from the AWS API. This
filtering is done locally on what AWS returns, and could have a performance
impact if the result is large. It is recommended to combine this with other
options to narrow down the list AWS returns.

~> **NOTE:** At least one of `executable_users`, `filter`, `owners`, or
`name_regex` must be specified.

~> **NOTE:** If more or less than a single match is returned by the search,
Terraform will fail. Ensure that your search is specific enough to return
a single AMI ID only, or use `most_recent` to choose the most recent one. If
you want to match multiple AMIs, use the `aws_ami_ids` data source instead.

## Attributes Reference

`id` is set to the ID of the found AMI. In addition, the following attributes
are exported:

~> **NOTE:** Some values are not always set and may not be available for
interpolation.

* `architecture` - The OS architecture of the AMI (ie: `i386` or `x86_64`).
* `block_device_mappings` - The block device mappings of the AMI.
  * `block_device_mappings.#.device_name` - The physical name of the device.
  * `block_device_mappings.#.ebs.delete_on_termination` - `true` if the EBS volume
    will be deleted on termination.
  * `block_device_mappings.#.ebs.encrypted` - `true` if the EBS volume
    is encrypted.
  * `block_device_mappings.#.ebs.iops` - `0` if the EBS volume is
    not a provisioned IOPS image, otherwise the supported IOPS count.
  * `block_device_mappings.#.ebs.snapshot_id` - The ID of the snapshot.
  * `block_device_mappings.#.ebs.volume_size` - The size of the volume, in GiB.
  * `block_device_mappings.#.ebs.volume_type` - The volume type.
  * `block_device_mappings.#.no_device` - Suppresses the specified device
    included in the block device mapping of the AMI.
  * `block_device_mappings.#.virtual_name` - The virtual device name (for
    instance stores).
* `creation_date` - The date and time the image was created.
* `description` - The description of the AMI that was provided during image
  creation.
* `hypervisor` - The hypervisor type of the image.
* `image_id` - The ID of the AMI. Should be the same as the resource `id`.
* `image_location` - The location of the AMI.
* `image_owner_alias` - The AWS account alias (for example, `amazon`, `self`) or
  the AWS account ID of the AMI owner.
* `image_type` - The type of image.
* `kernel_id` - The kernel associated with the image, if any. Only applicable
  for machine images.
* `name` - The name of the AMI that was provided during image creation.
* `owner_id` - The AWS account ID of the image owner.
* `platform` - The value is Windows for `Windows` AMIs; otherwise blank.
* `product_codes` - Any product codes associated with the AMI.
  * `product_codes.#.product_code_id` - The product code.
  * `product_codes.#.product_code_type` - The type of product code.
* `public` - `true` if the image has public launch permissions.
* `ramdisk_id` - The RAM disk associated with the image, if any. Only applicable
  for machine images.
* `root_device_name` - The device name of the root device.
* `root_device_type` - The type of root device (ie: `ebs` or `instance-store`).
* `sriov_net_support` - Specifies whether enhanced networking is enabled.
* `state` - The current state of the AMI. If the state is `available`, the image
  is successfully registered and can be used to launch an instance.
* `state_reason` - Describes a state change. Fields are `UNSET` if not available.
  * `state_reason.code` - The reason code for the state change.
  * `state_reason.message` - The message for the state change.
* `tags` - Any tags assigned to the image.
  * `tags.#.key` - The key name of the tag.
  * `tags.#.value` - The value of the tag.
* `virtualization_type` - The type of virtualization of the AMI (ie: `hvm` or
  `paravirtual`).

[1]: http://docs.aws.amazon.com/cli/latest/reference/ec2/describe-images.html
