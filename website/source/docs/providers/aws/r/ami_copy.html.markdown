---
layout: "aws"
page_title: "AWS: aws_ami_copy"
sidebar_current: "docs-aws-resource-ami-copy"
description: |-
  Duplicates an existing Amazon Machine Image (AMI)
---

# aws\_ami\_copy

The "AMI copy" resource allows duplication of an Amazon Machine Image (AMI),
including cross-region copies.

If the source AMI has associated EBS snapshots, those will also be duplicated
along with the AMI.

This is useful for taking a single AMI provisioned in one region and making
it available in another for a multi-region deployment.

Copying an AMI can take several minutes. The creation of this resource will
block until the new AMI is available for use on new instances.

## Example Usage

```
resource "aws_ami_copy" "example" {
    name = "terraform-example"
    description = "A copy of ami-xxxxxxxx"
    source_ami_id = "ami-xxxxxxxx"
    source_ami_region = "us-west-1"
    tags {
        Name = "HelloWorld"
    }
}
```

## Argument Reference

The following arguments are supported:

* `name` - (Required) A region-unique name for the AMI.
* `source_ami_id` - (Required) The id of the AMI to copy. This id must be valid in the region
  given by `source_ami_region`.
* `source_region` - (Required) The region from which the AMI will be copied. This may be the
  same as the AWS provider region in order to create a copy within the same region.

This resource also exposes the full set of arguments from the [`aws_ami`](ami.html) resource.

## Attributes Reference

The following attributes are exported:

* `id` - The ID of the created AMI.

This resource also exports a full set of attributes corresponding to the arguments of the
[`aws_ami`](ami.html) resource, allowing the properties of the created AMI to be used elsewhere in the
configuration.
