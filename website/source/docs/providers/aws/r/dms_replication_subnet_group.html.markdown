---
layout: "aws"
page_title: "AWS: aws_dms_replication_subnet_group"
sidebar_current: "docs-aws-resource-dms-replication-subnet-group"
description: |-
  Provides a DMS (Data Migration Service) subnet group resource.
---

# aws\_dms\_replication\_subnet\_group

Provides a DMS (Data Migration Service) replication subnet group resource. DMS replication subnet groups can be created, updated, deleted, and imported.

## Example Usage

```hcl
# Create a new replication subnet group
resource "aws_dms_replication_subnet_group" "test" {
  replication_subnet_group_description = "Test replication subnet group"
  replication_subnet_group_id          = "test-dms-replication-subnet-group-tf"

  subnet_ids = [
    "subnet-12345678",
  ]
}
```

## Argument Reference

The following arguments are supported:

* `replication_subnet_group_description` - (Required) The description for the subnet group.
* `replication_subnet_group_id` - (Required) The name for the replication subnet group. This value is stored as a lowercase string.

    - Must contain no more than 255 alphanumeric characters, periods, spaces, underscores, or hyphens.
    - Must not be "default".

* `subnet_ids` - (Required) A list of the EC2 subnet IDs for the subnet group.

## Attributes Reference

The following attributes are exported:

* `vpc_id` - The ID of the VPC the subnet group is in.

## Import

Replication subnet groups can be imported using the `replication_subnet_group_id`, e.g.

```
$ terraform import aws_dms_replication_subnet_group.test test-dms-replication-subnet-group-tf
```
