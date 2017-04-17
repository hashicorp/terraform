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

```hcl
resource "aws_redshift_security_group" "default" {
  name = "redshift-sg"

  ingress {
    cidr = "10.0.0.0/24"
  }
}
```

## Argument Reference

The following arguments are supported:

* `name` - (Required) The name of the Redshift security group.
* `description` - (Optional) The description of the Redshift security group. Defaults to "Managed by Terraform".
* `ingress` - (Optional) A list of ingress rules.

Ingress blocks support the following:

* `cidr` - The CIDR block to accept
* `security_group_name` - The name of the security group to authorize
* `security_group_owner_id` - The owner Id of the security group provided
  by `security_group_name`.

## Attributes Reference

The following attributes are exported:

* `id` - The Redshift security group ID.

## Import

Redshift security groups can be imported using the `name`, e.g.

```
$ terraform import aws_redshift_security_group.testgroup1 redshift_test_group
```
