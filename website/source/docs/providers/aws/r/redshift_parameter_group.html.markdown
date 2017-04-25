---
layout: "aws"
page_title: "AWS: aws_redshift_parameter_group"
sidebar_current: "docs-aws-resource-redshift-parameter-group"
---

# aws\_redshift\_parameter\_group

Provides a Redshift Cluster parameter group resource.

## Example Usage

```hcl
resource "aws_redshift_parameter_group" "bar" {
  name   = "parameter-group-test-terraform"
  family = "redshift-1.0"

  parameter {
    name  = "require_ssl"
    value = "true"
  }

  parameter {
    name  = "query_group"
    value = "example"
  }

  parameter {
    name  = "enable_user_activity_logging"
    value = "true"
  }
}
```

## Argument Reference

The following arguments are supported:

* `name` - (Required) The name of the Redshift parameter group.
* `family` - (Required) The family of the Redshift parameter group.
* `description` - (Optional) The description of the Redshift parameter group. Defaults to "Managed by Terraform".
* `parameter` - (Optional) A list of Redshift parameters to apply.

Parameter blocks support the following:

* `name` - (Required) The name of the Redshift parameter.
* `value` - (Required) The value of the Redshift parameter.

You can read more about the parameters that Redshift supports in the [documentation](http://docs.aws.amazon.com/redshift/latest/mgmt/working-with-parameter-groups.html)

## Attributes Reference

The following attributes are exported:

* `id` - The Redshift parameter group name.

## Import

Redshift Parameter Groups can be imported using the `name`, e.g.

```
$ terraform import aws_redshift_parameter_group.paramgroup1 parameter-group-test-terraform
```