---
layout: "aws"
page_title: "AWS: aws_redshift_security_group"
sidebar_current: "docs-aws-resource-redshift-security-group"
description: |-
  Provides a Redshift security group resource.
---

# aws\_redshift\_security\_group

Creates a new Amazon Redshift security group. You use security groups to control access to non-VPC clusters

## Example Usage

```
resource "aws_redshift_security_group" "default" {
    name = "redshift_sg"
    description = "Redshift Example security group"

    ingress {
        cidr = "10.0.0.0/24"
    }
}
```

## Argument Reference

The following arguments are supported:

* `name` - (Required) The name of the Redshift security group.
* `description` - (Required) The description of the Redshift security group.
* `ingress` - (Optional) A list of ingress rules.

Ingress blocks support the following:

* `cidr` - The CIDR block to accept
* `security_group_name` - The name of the security group to authorize
* `security_group_owner_id` - The owner Id of the security group provided
  by `security_group_name`.

## Attributes Reference

The following attributes are exported:

* `id` - The Redshift security group ID.

