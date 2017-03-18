---
layout: "aws"
page_title: "AWS: aws_instances"
sidebar_current: "docs-aws-datasource-instances"
description: |-
  Get information on multiple Amazon EC2 Instances.
---

# aws\_instances

Use this data source to get the attributes of multiple Amazon EC2 Instances for use in other
resources.

## Example Usage

```
data "aws_instances" "foo" {
  instance_ids = ["i-instanceid1"]

  filter {
    name   = "image-id"
    values = ["ami-xxxxxxxx"]
  }

  filter {
    name   = "tag:Name"
    values = ["instance-name-tag"]
  }
}
```

## Argument Reference

* `instance_ids` - (Optional) Specify list of Instance IDs with which to populate the data source.

* `instance_tags` - (Optional) A mapping of tags, each pair of which must
exactly match a pair on the desired Instances.

* `filter` - (Optional) One or more name/value pairs to use as filters. There are
several valid keys, for a full reference, check out
[describe-instances in the AWS CLI reference][1].

~> **NOTE:** At least one of `filter`, `instance_tags`, or `instance_ids` must be specified.

~> **NOTE:** Unlike the `aws_instance` data source, the search arguments used with `aws_instances`
may match 0 or more instances.

## Attributes Reference

All of the argument attributes are also exported as result attributes. This data source will
complete the data by populating any fields that are not included in the configuration with
the data for the selected AWS Instances.

~> **NOTE:** Some values are not always set and may not be available for
interpolation.

* `instances` - The attributes of all the AWS Instances matched by the provided search arguments.
Refer to documenation for [aws_instance](/docs/providers/aws/d/instance.html) to view attributes
that should be available on each instance.

[1]: http://docs.aws.amazon.com/cli/latest/reference/ec2/describe-instances.html
