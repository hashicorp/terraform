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

```
resource "aws_db_subnet_group" "default" {
    name = "main"
    description = "Our main group of subnets"
    subnet_ids = ["${aws_subnet.frontend.id}", "${aws_subnet.backend.id}"]
}
```

## Argument Reference

The following arguments are supported:

* `name` - (Required) The name of the DB subnet group.
* `description` - (Required) The description of the DB subnet group.
* `subnet_ids` - (Required) A list of VPC subnet IDs.

## Attributes Reference

The following attributes are exported:

* `id` - The db subnet group name.

