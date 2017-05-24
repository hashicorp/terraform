---
layout: "aws"
page_title: "AWS: aws_opsworks_rds_db_instance"
sidebar_current: "docs-aws-resource-opsworks-rds-db-instance"
description: |-
  Provides an OpsWorks RDS DB Instance resource.
---

# aws\_opsworks\_rds\_db\_instance

Provides an OpsWorks RDS DB Instance resource.

~> **Note:** All arguments including the username and password will be stored in the raw state as plain-text.
[Read more about sensitive data in state](/docs/state/sensitive-data.html).

## Example Usage

```hcl
resource "aws_opsworks_rds_db_instance" "my_instance" {
  stack_id            = "${aws_opsworks_stack.my_stack.id}"
  rds_db_instance_arn = "${aws_db_instance.my_instance.arn}"
  db_user             = "someUser"
  db_password         = "somePass"
}
```

## Argument Reference

The following arguments are supported:

* `stack_id` - (Required) The stack to register a db inatance for. Changing this will force a new resource.
* `rds_db_instance_arn` - (Required) The db instance to register for this stack. Changing this will force a new resource.
* `db_user` - (Required) A db username
* `db_password` - (Required) A db password

## Attributes Reference

The following attributes are exported:

* `id` - The computed id. Please note that this is only used internally to identify the stack <-> instance relation. This value is not used in aws.
