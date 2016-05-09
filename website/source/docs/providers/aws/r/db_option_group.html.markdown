---
layout: "aws"
page_title: "AWS: aws_db_option_group"
sidebar_current: "docs-aws-resource-db-option-group"
---

# aws\_db\_option\_group

Provides an RDS DB option group resource.

## Example Usage

```
resource "aws_db_option_group" "bar" {
  name = "option-group-test-terraform"
  option_group_description = "Terraform Option Group"
  engine_name = "sqlserver-ee"
  major_engine_version = "11.00"

  option {
	option_name = "mirroring"
  }

  option {
 	option_name = "TDE"
  }
	
  apply_immediately = true
}
```

## Argument Reference

The following arguments are supported:

* `name` - (Required) The name of the Option group to be created.
* `option_group_description` - (Required) The description of the option group.
* `engine_name` - (Required) Specifies the name of the engine that this option group should be associated with..
* `major_engine_version` - (Required) Specifies the major version of the engine that this option group should be associated with.
* `option` - (Optional) A list of Options to apply.
* `tags` - (Optional) A mapping of tags to assign to the resource.

Option blocks support the following:

* `option_name` - (Required) The Name of the Option (e.g. MEMCACHED).
* `port` - (Optional) The Port number when connecting to the Option (e.g. 11211).
* `db_security_group_memberships` - (Optional) A list of DB Security Groups for which the option is enabled.
* `vpc_security_group_memberships` - (Optional) A list of VPC Security Groups for which the option is enabled.

## Attributes Reference

The following attributes are exported:

* `arn` - The ARN of the db option group.
