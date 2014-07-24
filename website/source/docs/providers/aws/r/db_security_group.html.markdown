---
layout: "aws"
page_title: "AWS: aws_db_security_group"
sidebar_current: "docs-aws-resource-db-security-group"
---

# aws\_db\_security\_group

Provides an RDS security group resource.

## Example Usage

```
resource "aws_db_security_group" "default" {
    name = "RDS default security group"
    ingress {
        cidr = "10.0.0.1/24"
    }
}
```

## Argument Reference

The following arguments are supported:

* `name` - (Required) The name of the DB security group.
* `description` - (Required) The description of the DB security group.
* `ingress` - (Optional) A list of ingress rules.

Ingress blocks support the following:

* `cidr` - The CIDR block to accept
* `security_group_name` - The name of the security group to authorize
* `security_group_id` - The ID of the security group to authorize
* `security_group_owner_id` - The owner Id of the security group provided
  by `security_group_name`.

## Attributes Reference

The following attributes are exported:

* `id` - The db security group ID.

