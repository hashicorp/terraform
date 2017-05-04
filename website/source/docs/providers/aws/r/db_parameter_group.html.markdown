---
layout: "aws"
page_title: "AWS: aws_db_parameter_group"
sidebar_current: "docs-aws-resource-db-parameter-group"
---

# aws\_db\_parameter\_group

Provides an RDS DB parameter group resource.

## Example Usage

```hcl
resource "aws_db_parameter_group" "default" {
  name   = "rds-pg"
  family = "mysql5.6"

  parameter {
    name  = "character_set_server"
    value = "utf8"
  }

  parameter {
    name  = "character_set_client"
    value = "utf8"
  }
}
```

## Argument Reference

The following arguments are supported:

* `name` - (Optional, Forces new resource) The name of the DB parameter group. If omitted, Terraform will assign a random, unique name.
* `name_prefix` - (Optional, Forces new resource) Creates a unique name beginning with the specified prefix. Conflicts with `name`.
* `family` - (Required) The family of the DB parameter group.
* `description` - (Optional) The description of the DB parameter group. Defaults to "Managed by Terraform".
* `parameter` - (Optional) A list of DB parameters to apply.
* `tags` - (Optional) A mapping of tags to assign to the resource.

Parameter blocks support the following:

* `name` - (Required) The name of the DB parameter.
* `value` - (Required) The value of the DB parameter.
* `apply_method` - (Optional) "immediate" (default), or "pending-reboot". Some
    engines can't apply some parameters without a reboot, and you will need to
    specify "pending-reboot" here.

## Attributes Reference

The following attributes are exported:

* `id` - The db parameter group name.
* `arn` - The ARN of the db parameter group.

## Import

DB Parameter groups can be imported using the `name`, e.g.

```
$ terraform import aws_db_parameter_group.rds_pg rds-pg
```
