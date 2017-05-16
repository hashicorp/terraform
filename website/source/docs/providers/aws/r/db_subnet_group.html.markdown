---
layout: "aws"
page_title: "AWS: aws_db_subnet_group"
sidebar_current: "docs-aws-resource-db-subnet-group"
description: |-
  Provides an RDS DB subnet group resource.
---

# aws\_db\_subnet\_group

Provides an RDS DB subnet group resource.

## Example Usage

```hcl
resource "aws_db_subnet_group" "default" {
  name       = "main"
  subnet_ids = ["${aws_subnet.frontend.id}", "${aws_subnet.backend.id}"]

  tags {
    Name = "My DB subnet group"
  }
}
```

## Argument Reference

The following arguments are supported:

* `name` - (Optional, Forces new resource) The name of the DB subnet group. If omitted, Terraform will assign a random, unique name.
* `name_prefix` - (Optional, Forces new resource) Creates a unique name beginning with the specified prefix. Conflicts with `name`.
* `description` - (Optional) The description of the DB subnet group. Defaults to "Managed by Terraform".
* `subnet_ids` - (Required) A list of VPC subnet IDs.
* `tags` - (Optional) A mapping of tags to assign to the resource.

## Attributes Reference

The following attributes are exported:

* `id` - The db subnet group name.
* `arn` - The ARN of the db subnet group.


## Import

DB Subnet groups can be imported using the `name`, e.g.

```
$ terraform import aws_db_subnet_group.default production-subnet-group
```
