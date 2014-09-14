---
layout: "aws"
page_title: "AWS: aws_db_subnet_group"
sidebar_current: "docs-aws-resource-db-subnet-group"
---

# aws\_db\_subnet\_group

Provides an RDS DB subnet group resource.

## Example Usage

```
resource "aws_db_subnet_group" "default" {
    name = "default"
    description = "RDS default subnet group"
    subnet_ids = ["${aws_subnet.foo.id}", "${aws_subnet.bar.id}"]
}
```

## Argument Reference

The following arguments are supported:

* `name` - (Required) The name of the DB subnet group.
* `description` - (Required) The description of the DB subnet group.
* `subnet_ids` - (Required) A list of subnet IDs.

## Attributes Reference

The following attributes are exported:

* `name` - The DB subnet group name.
* `description` - The DB subnet group description.
