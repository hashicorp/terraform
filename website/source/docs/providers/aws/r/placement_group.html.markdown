---
layout: "aws"
page_title: "AWS: aws_placement_group"
sidebar_current: "docs-aws-resource-placement-group"
description: |-
  Provides an EC2 placement group.
---

# aws\_placement\_group

Provides an EC2 placement group. Read more about placement groups
in [AWS Docs](http://docs.aws.amazon.com/AWSEC2/latest/UserGuide/placement-groups.html).

## Example Usage

```
resource "aws_placement_group" "web" {
    name = "hunky-dory-pg"
    strategy = "cluster"
}
```

## Argument Reference

The following arguments are supported:

* `name` - (Required) The name of the placement group.
* `strategy` - (Required) The placement strategy. The only supported value is `cluster`

## Attributes Reference

The following attributes are exported:

* `id` - The name of the placement group.
