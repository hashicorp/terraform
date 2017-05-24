---
layout: "aws"
page_title: "AWS: aws_db_instance"
sidebar_current: "docs-aws-datasource-db-instance"
description: |-
  Get information on an RDS Database Instance.
---

# aws\_db\_instance

Use this data source to get information about an RDS instance

## Example Usage

```hcl
data "aws_db_instance" "database" {
  db_instance_identifier = "my-test-database"
}
```

## Argument Reference

The following arguments are supported:

* `db_instance_identifier` - (Required) The name of the RDS instance

## Attributes Reference

The following attributes are exported:

* `address` - The address of the RDS instance.
* `allocated_storage` - Specifies the allocated storage size specified in gigabytes.
* `auto_minor_version_upgrade` - Indicates that minor version patches are applied automatically.
* `availability_zone` - Specifies the name of the Availability Zone the DB instance is located in.
* `backup_retention_period` - Specifies the number of days for which automatic DB snapshots are retained.
* `db_cluster_identifier` - If the DB instance is a member of a DB cluster, contains the name of the DB cluster that the DB instance is a member of.
* `db_instance_arn` - The Amazon Resource Name (ARN) for the DB instance.
* `db_instance_class` - Contains the name of the compute and memory capacity class of the DB instance.
* `db_name` - Contains the name of the initial database of this instance that was provided at create time, if one was specified when the DB instance was created. This same name is returned for the life of the DB instance.
* `db_parameter_groups` - Provides the list of DB parameter groups applied to this DB instance.
* `db_security_groups` - Provides List of DB security groups associated to this DB instance.
* `db_subnet_group` - Specifies the name of the subnet group associated with the DB instance.
* `db_instance_port` - Specifies the port that the DB instance listens on.
* `endpoint` - The connection endpoint.
* `engine` - Provides the name of the database engine to be used for this DB instance.
* `engine_version` - Indicates the database engine version.
* `hosted_zone_id` - The canonical hosted zone ID of the DB instance (to be used in a Route 53 Alias record).
* `iops` - Specifies the Provisioned IOPS (I/O operations per second) value.
* `kms_key_id` - If StorageEncrypted is true, the KMS key identifier for the encrypted DB instance.
* `license_model` - License model information for this DB instance.
* `master_username` - Contains the master username for the DB instance.
* `monitoring_interval` - The interval, in seconds, between points when Enhanced Monitoring metrics are collected for the DB instance.
* `monitoring_role_arn` - The ARN for the IAM role that permits RDS to send Enhanced Monitoring metrics to CloudWatch Logs.
* `multi_az` - Specifies if the DB instance is a Multi-AZ deployment.
* `option_group_memberships` - Provides the list of option group memberships for this DB instance.
* `port` - The database port.
* `preferred_backup_window` - Specifies the daily time range during which automated backups are created.
* `preferred_maintenance_window` -  Specifies the weekly time range during which system maintenance can occur in UTC.
* `publicly_accessible` - Specifies the accessibility options for the DB instance.
* `storage_encrypted` - Specifies whether the DB instance is encrypted.
* `storage_type` - Specifies the storage type associated with DB instance.
* `timezone` - The time zone of the DB instance.
* `vpc_security_groups` - Provides a list of VPC security group elements that the DB instance belongs to.
* `replicate_source_db` - The identifier of the source DB that this is a replica of.
