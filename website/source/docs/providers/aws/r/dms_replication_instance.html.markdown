---
layout: "aws"
page_title: "AWS: aws_dms_replication_instance"
sidebar_current: "docs-aws-resource-dms-replication-instance"
description: |-
  Provides a DMS (Data Migration Service) replication instance resource.
---

# aws\_dms\_replication\_instance

Provides a DMS (Data Migration Service) replication instance resource. DMS replication instances can be created, updated, deleted, and imported.

## Example Usage

```hcl
# Create a new replication instance
resource "aws_dms_replication_instance" "test" {
  allocated_storage            = 20
  apply_immediately            = true
  auto_minor_version_upgrade   = true
  availability_zone            = "us-west-2c"
  engine_version               = "1.9.0"
  kms_key_arn                  = "arn:aws:kms:us-east-1:123456789012:key/12345678-1234-1234-1234-123456789012"
  multi_az                     = false
  preferred_maintenance_window = "sun:10:30-sun:14:30"
  publicly_accessible          = true
  replication_instance_class   = "dms.t2.micro"
  replication_instance_id      = "test-dms-replication-instance-tf"
  replication_subnet_group_id  = "${aws_dms_replication_subnet_group.test-dms-replication-subnet-group-tf}"

  tags {
    Name = "test"
  }

  vpc_security_group_ids = [
    "sg-12345678",
  ]
}
```

## Argument Reference

The following arguments are supported:

* `allocated_storage` - (Optional, Default: 50, Min: 5, Max: 6144) The amount of storage (in gigabytes) to be initially allocated for the replication instance.
* `apply_immediately` - (Optional, Default: false) Indicates whether the changes should be applied immediately or during the next maintenance window. Only used when updating an existing resource.
* `auto_minor_version_upgrade` - (Optional, Default: false) Indicates that minor engine upgrades will be applied automatically to the replication instance during the maintenance window.
* `availability_zone` - (Optional) The EC2 Availability Zone that the replication instance will be created in.
* `engine_version` - (Optional) The engine version number of the replication instance.
* `kms_key_arn` - (Optional) The Amazon Resource Name (ARN) for the KMS key that will be used to encrypt the connection parameters. If you do not specify a value for `kms_key_arn`, then AWS DMS will use your default encryption key. AWS KMS creates the default encryption key for your AWS account. Your AWS account has a different default encryption key for each AWS region.
* `multi_az` - (Optional) Specifies if the replication instance is a multi-az deployment. You cannot set the `availability_zone` parameter if the `multi_az` parameter is set to `true`.
* `preferred_maintenance_window` - (Optional) The weekly time range during which system maintenance can occur, in Universal Coordinated Time (UTC).

    - Default: A 30-minute window selected at random from an 8-hour block of time per region, occurring on a random day of the week.
    - Format: `ddd:hh24:mi-ddd:hh24:mi`
    - Valid Days: `mon, tue, wed, thu, fri, sat, sun`
    - Constraints: Minimum 30-minute window.

* `publicly_accessible` - (Optional, Default: false) Specifies the accessibility options for the replication instance. A value of true represents an instance with a public IP address. A value of false represents an instance with a private IP address.
* `replication_instance_class` - (Required) The compute and memory capacity of the replication instance as specified by the replication instance class. Can be one of `dms.t2.micro | dms.t2.small | dms.t2.medium | dms.t2.large | dms.c4.large | dms.c4.xlarge | dms.c4.2xlarge | dms.c4.4xlarge`
* `replication_instance_id` - (Required) The replication instance identifier. This parameter is stored as a lowercase string.

    - Must contain from 1 to 63 alphanumeric characters or hyphens.
    - First character must be a letter.
    - Cannot end with a hyphen
    - Cannot contain two consecutive hyphens.

* `replication_subnet_group_id` - (Optional) A subnet group to associate with the replication instance.
* `tags` - (Optional) A mapping of tags to assign to the resource.
* `vpc_security_group_ids` - (Optional) A list of VPC security group IDs to be used with the replication instance. The VPC security groups must work with the VPC containing the replication instance.

## Attributes Reference

The following attributes are exported:

* `replication_instance_arn` - The Amazon Resource Name (ARN) of the replication instance.
* `replication_instance_private_ips` -  A list of the private IP addresses of the replication instance.
* `replication_instance_public_ips` - A list of the public IP addresses of the replication instance.

<a id="timeouts"></a>
## Timeouts

`aws_dms_replication_instance` provides the following
[Timeouts](/docs/configuration/resources.html#timeouts) configuration options:

- `create` - (Default `30 minutes`) Used for Creating Instances
- `update` - (Default `30 minutes`) Used for Database modifications
- `delete` - (Default `30 minutes`) Used for destroying databases.

## Import

Replication instances can be imported using the `replication_instance_id`, e.g.

```
$ terraform import aws_dms_replication_instance.test test-dms-replication-instance-tf
```
