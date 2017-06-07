---
layout: "aws"
page_title: "AWS: aws_db_option_group"
sidebar_current: "docs-aws-resource-db-option-group"
---

# aws\_db\_option\_group

Provides an RDS DB option group resource.

## Example Usage

```hcl
resource "aws_db_option_group" "bar" {
  name                     = "option-group-test-terraform"
  option_group_description = "Terraform Option Group"
  engine_name              = "sqlserver-ee"
  major_engine_version     = "11.00"

  option {
    option_name = "Timezone"

    option_settings {
      name  = "TIME_ZONE"
      value = "UTC"
    }
  }

  option {
    option_name = "TDE"
  }
}
```

~> **Note**: Any modifications to the `db_option_group` are set to happen immediately as we default to applying immediately.

## Argument Reference

The following arguments are supported:

* `name` - (Optional, Forces new resource) The name of the option group. If omitted, Terraform will assign a random, unique name. Must be lowercase, to match as it is stored in AWS.
* `name_prefix` - (Optional, Forces new resource) Creates a unique name beginning with the specified prefix. Conflicts with `name`. Must be lowercase, to match as it is stored in AWS.
* `option_group_description` - (Optional) The description of the option group. Defaults to "Managed by Terraform".
* `engine_name` - (Required) Specifies the name of the engine that this option group should be associated with.
* `major_engine_version` - (Required) Specifies the major version of the engine that this option group should be associated with.
* `option` - (Optional) A list of Options to apply.
* `tags` - (Optional) A mapping of tags to assign to the resource.

Option blocks support the following:

* `option_name` - (Required) The Name of the Option (e.g. MEMCACHED).
* `option_settings` - (Optional) A list of option settings to apply.
* `port` - (Optional) The Port number when connecting to the Option (e.g. 11211).
* `db_security_group_memberships` - (Optional) A list of DB Security Groups for which the option is enabled.
* `vpc_security_group_memberships` - (Optional) A list of VPC Security Groups for which the option is enabled.

Option Settings blocks support the following:

* `name` - (Optional) The Name of the setting.
* `value` - (Optional) The Value of the setting.

## Attributes Reference

The following attributes are exported:

* `arn` - The ARN of the db option group.

<a id="timeouts"></a>
## Timeouts

`aws_db_option_group` provides the following
[Timeouts](/docs/configuration/resources.html#timeouts) configuration options:

- `delete` - (Default `15 minutes`)

## Import

DB Option groups can be imported using the `name`, e.g.

```
$ terraform import aws_db_option_group.bar mysql-option-group
```
