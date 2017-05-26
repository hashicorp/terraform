---
layout: "aws"
page_title: "AWS: aws_db_security_group"
sidebar_current: "docs-aws-resource-db-security-group"
description: |-
  Provides an RDS security group resource.
---

# aws\_db\_security\_group

Provides an RDS security group resource. This is only for DB instances in the
EC2-Classic Platform. For instances inside a VPC, use the
[`aws_db_instance.vpc_security_group_ids`](/docs/providers/aws/r/db_instance.html#vpc_security_group_ids)
attribute instead.

## Example Usage

```hcl
resource "aws_db_security_group" "default" {
  name = "rds_sg"

  ingress {
    cidr = "10.0.0.0/24"
  }
}
```

## Argument Reference

The following arguments are supported:

* `name` - (Required) The name of the DB security group.
* `description` - (Optional) The description of the DB security group. Defaults to "Managed by Terraform".
* `ingress` - (Required) A list of ingress rules.
* `tags` - (Optional) A mapping of tags to assign to the resource.

Ingress blocks support the following:

* `cidr` - The CIDR block to accept
* `security_group_name` - The name of the security group to authorize
* `security_group_id` - The ID of the security group to authorize
* `security_group_owner_id` - The owner Id of the security group provided
  by `security_group_name`.

## Attributes Reference

The following attributes are exported:

* `id` - The db security group ID.
* `arn` - The arn of the DB security group.


## Import

DB Security groups can be imported using the `name`, e.g.

```
$ terraform import aws_db_security_group.default aws_rds_sg-1
```
