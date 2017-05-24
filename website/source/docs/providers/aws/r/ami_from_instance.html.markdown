---
layout: "aws"
page_title: "AWS: aws_ami_from_instance"
sidebar_current: "docs-aws-resource-ami-from-instance"
description: |-
  Creates an Amazon Machine Image (AMI) from an EBS-backed EC2 instance
---

# aws\_ami\_from\_instance

The "AMI from instance" resource allows the creation of an Amazon Machine
Image (AMI) modelled after an existing EBS-backed EC2 instance.

The created AMI will refer to implicitly-created snapshots of the instance's
EBS volumes and mimick its assigned block device configuration at the time
the resource is created.

This resource is best applied to an instance that is stopped when this instance
is created, so that the contents of the created image are predictable. When
applied to an instance that is running, *the instance will be stopped before taking
the snapshots and then started back up again*, resulting in a period of
downtime.

Note that the source instance is inspected only at the initial creation of this
resource. Ongoing updates to the referenced instance will not be propagated into
the generated AMI. Users may taint or otherwise recreate the resource in order
to produce a fresh snapshot.

## Example Usage

```hcl
resource "aws_ami_from_instance" "example" {
  name               = "terraform-example"
  source_instance_id = "i-xxxxxxxx"
}
```

## Argument Reference

The following arguments are supported:

* `name` - (Required) A region-unique name for the AMI.
* `source_instance_id` - (Required) The id of the instance to use as the basis of the AMI.
* `snapshot_without_reboot` - (Optional) Boolean that overrides the behavior of stopping
  the instance before snapshotting. This is risky since it may cause a snapshot of an
  inconsistent filesystem state, but can be used to avoid downtime if the user otherwise
  guarantees that no filesystem writes will be underway at the time of snapshot.

## Attributes Reference

The following attributes are exported:

* `id` - The ID of the created AMI.

This resource also exports a full set of attributes corresponding to the arguments of the
`aws_ami` resource, allowing the properties of the created AMI to be used elsewhere in the
configuration.
