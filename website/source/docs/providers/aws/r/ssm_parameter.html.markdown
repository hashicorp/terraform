---
layout: "aws"
page_title: "AWS: aws_ssm_parameter"
sidebar_current: "docs-aws-resource-ssm-parameter"
description: |-
  Provides a SSM Parameter resource
---

# aws\_ssm\_parameter

Provides an SSM Parameter resource.

## Example Usage

To store a basic string parameter:

```hcl
resource "aws_ssm_parameter" "foo" {
  name  = "foo"
  type  = "String"
  value = "bar"
}
```

To store an encrypted string using the default SSM KMS key:

```hcl
resource "aws_db_instance" "default" {
  allocated_storage    = 10
  storage_type         = "gp2"
  engine               = "mysql"
  engine_version       = "5.7.16"
  instance_class       = "db.t2.micro"
  name                 = "mydb"
  username             = "foo"
  password             = "${var.database_master_password}"
  db_subnet_group_name = "my_database_subnet_group"
  parameter_group_name = "default.mysql5.7"
}

resource "aws_ssm_parameter" "secret" {
  name  = "${var.environment}/database/password/master"
  type  = "SecureString"
  value = "${var.database_master_password}"
}
```

~> **Note:** The unencrypted value of a SecureString will be stored in the raw state as plain-text.
[Read more about sensitive data in state](/docs/state/sensitive-data.html).

## Argument Reference

The following arguments are supported:

* `name` - (Required) The name of the parameter.
* `type` - (Required) The type of the parameter. Valid types are `String`, `StringList` and `SecureString`.
* `value` - (Required) The value of the parameter.
* `key_id` - (Optional) The KMS key id or arn for encrypting a SecureString.
## Attributes Reference

The following attributes are exported:

* `name` - (Required) The name of the parameter.
* `type` - (Required) The type of the parameter. Valid types are `String`, `StringList` and `SecureString`.
* `value` - (Required) The value of the parameter.
