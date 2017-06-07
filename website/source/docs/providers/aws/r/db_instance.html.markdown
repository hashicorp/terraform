---
layout: "aws"
page_title: "AWS: aws_db_instance"
sidebar_current: "docs-aws-resource-db-instance"
description: |-
  Provides an RDS instance resource.
---

# aws\_db\_instance

Provides an RDS instance resource.  A DB instance is an isolated database
environment in the cloud.  A DB instance can contain multiple user-created
databases.

Changes to a DB instance can occur when you manually change a
parameter, such as `allocated_storage`, and are reflected in the next maintenance
window. Because of this, Terraform may report a difference in its planning
phase because a modification has not yet taken place. You can use the
`apply_immediately` flag to instruct the service to apply the change immediately
(see documentation below).

When upgrading the major version of an engine, `allow_major_version_upgrade` must be set to `true`

~> **Note:** using `apply_immediately` can result in a
brief downtime as the server reboots. See the AWS Docs on [RDS Maintenance][2]
for more information.

~> **Note:** All arguments including the username and password will be stored in the raw state as plain-text.
[Read more about sensitive data in state](/docs/state/sensitive-data.html).

## Example Usage

```hcl
resource "aws_db_instance" "default" {
  allocated_storage    = 10
  storage_type         = "gp2"
  engine               = "mysql"
  engine_version       = "5.6.17"
  instance_class       = "db.t1.micro"
  name                 = "mydb"
  username             = "foo"
  password             = "bar"
  db_subnet_group_name = "my_database_subnet_group"
  parameter_group_name = "default.mysql5.6"
}
```

## Argument Reference

For more detailed documentation about each argument, refer to
the [AWS official documentation](http://docs.aws.amazon.com/AmazonRDS/latest/APIReference/API_CreateDBInstance.html).

The following arguments are supported:

* `allocated_storage` - (Required unless a `snapshot_identifier` or `replicate_source_db` is provided) The allocated storage in gigabytes.
* `engine` - (Required unless a `snapshot_identifier` or `replicate_source_db` is provided) The database engine to use.
* `engine_version` - (Optional) The engine version to use.
* `identifier` - (Optional, Forces new resource) The name of the RDS instance, if omitted, Terraform will assign a random, unique identifier.
* `identifier_prefix` - (Optional, Forces new resource) Creates a unique identifier beginning with the specified prefix. Conflicts with `identifer`.
* `instance_class` - (Required) The instance type of the RDS instance.
* `storage_type` - (Optional) One of "standard" (magnetic), "gp2" (general
    purpose SSD), or "io1" (provisioned IOPS SSD). The default is "io1" if
    `iops` is specified, "standard" if not. Note that this behaviour is different from the AWS web console, where the default is "gp2".
* `final_snapshot_identifier` - (Optional) The name of your final DB snapshot
    when this DB instance is deleted. If omitted, no final snapshot will be
    made.
* `skip_final_snapshot` - (Optional) Determines whether a final DB snapshot is created before the DB instance is deleted. If true is specified, no DBSnapshot is created. If false is specified, a DB snapshot is created before the DB instance is deleted, using the value from `final_snapshot_identifier`. Default is `false`.
* `copy_tags_to_snapshot` – (Optional, boolean) On delete, copy all Instance `tags` to
the final snapshot (if `final_snapshot_identifier` is specified). Default
`false`
* `name` - (Optional) The DB name to create. If omitted, no database is created
    initially.
* `password` - (Required unless a `snapshot_identifier` or `replicate_source_db` is provided) Password for the master DB user. Note that this may
    show up in logs, and it will be stored in the state file.
* `username` - (Required unless a `snapshot_identifier` or `replicate_source_db` is provided) Username for the master DB user.
* `availability_zone` - (Optional) The AZ for the RDS instance.
* `backup_retention_period` - (Optional) The days to retain backups for. Must be
`1` or greater to be a source for a [Read Replica][1].
* `backup_window` - (Optional) The daily time range (in UTC) during which automated backups are created if they are enabled. Example: "09:46-10:16". Must not overlap with `maintenance_window`.
* `iops` - (Optional) The amount of provisioned IOPS. Setting this implies a
    storage_type of "io1".
* `maintenance_window` - (Optional) The window to perform maintenance in.
  Syntax: "ddd:hh24:mi-ddd:hh24:mi". Eg: "Mon:00:00-Mon:03:00".
  See [RDS Maintenance Window docs](http://docs.aws.amazon.com/AmazonRDS/latest/UserGuide/USER_UpgradeDBInstance.Maintenance.html#AdjustingTheMaintenanceWindow) for more.
* `multi_az` - (Optional) Specifies if the RDS instance is multi-AZ
* `port` - (Optional) The port on which the DB accepts connections.
* `publicly_accessible` - (Optional) Bool to control if instance is publicly accessible. Defaults to `false`.
* `vpc_security_group_ids` - (Optional) List of VPC security groups to associate.
* `security_group_names` - (Optional/Deprecated) List of DB Security Groups to associate.
    Only used for [DB Instances on the _EC2-Classic_ Platform](https://docs.aws.amazon.com/AmazonRDS/latest/UserGuide/USER_VPC.html#USER_VPC.FindDefaultVPC).
* `db_subnet_group_name` - (Optional) Name of DB subnet group. DB instance will be created in the VPC associated with the DB subnet group. If unspecified, will be created in the `default` VPC, or in EC2 Classic, if available.
* `parameter_group_name` - (Optional) Name of the DB parameter group to associate.
* `option_group_name` - (Optional) Name of the DB option group to associate.
* `storage_encrypted` - (Optional) Specifies whether the DB instance is encrypted. The default is `false` if not specified.
* `apply_immediately` - (Optional) Specifies whether any database modifications
     are applied immediately, or during the next maintenance window. Default is
     `false`. See [Amazon RDS Documentation for more information.](https://docs.aws.amazon.com/AmazonRDS/latest/UserGuide/Overview.DBInstance.Modifying.html)
* `replicate_source_db` - (Optional) Specifies that this resource is a Replicate
database, and to use this value as the source database. This correlates to the
`identifier` of another Amazon RDS Database to replicate. See
[DB Instance Replication][1] and
[Working with PostgreSQL and MySQL Read Replicas](https://docs.aws.amazon.com/AmazonRDS/latest/UserGuide/USER_ReadRepl.html) for
 more information on using Replication.
* `snapshot_identifier` - (Optional) Specifies whether or not to create this database from a snapshot. This correlates to the snapshot ID you'd find in the RDS console, e.g: rds:production-2015-06-26-06-05.
* `license_model` - (Optional, but required for some DB engines, i.e. Oracle SE1) License model information for this DB instance.
* `auto_minor_version_upgrade` - (Optional) Indicates that minor engine upgrades will be applied automatically to the DB instance during the maintenance window. Defaults to true.
* `allow_major_version_upgrade` - (Optional) Indicates that major version upgrades are allowed. Changing this parameter does not result in an outage and the change is asynchronously applied as soon as possible.
* `monitoring_role_arn` - (Optional) The ARN for the IAM role that permits RDS to send
enhanced monitoring metrics to CloudWatch Logs. You can find more information on the [AWS Documentation](https://docs.aws.amazon.com/AmazonRDS/latest/UserGuide/USER_Monitoring.html)
what IAM permissions are needed to allow Enhanced Monitoring for RDS Instances.
* `monitoring_interval` - (Optional) The interval, in seconds, between points when Enhanced Monitoring metrics are collected for the DB instance. To disable collecting Enhanced Monitoring metrics, specify 0. The default is 0. Valid Values: 0, 1, 5, 10, 15, 30, 60.
* `kms_key_id` - (Optional) The ARN for the KMS encryption key.
* `character_set_name` - (Optional) The character set name to use for DB encoding in Oracle instances. This can't be changed.
[Oracle Character Sets Supported in Amazon RDS](https://docs.aws.amazon.com/AmazonRDS/latest/UserGuide/Appendix.OracleCharacterSets.html)
* `iam_database_authentication_enabled` - (Optional) Specifies whether or mappings of AWS Identity and Access Management (IAM) accounts to database accounts is enabled.
* `tags` - (Optional) A mapping of tags to assign to the resource.
* `timezone` - (Optional) Time zone of the DB instance. `timezone` is currently only supported by Microsoft SQL Server.
The `timezone` can only be set on creation. See [MSSQL User Guide](http://docs.aws.amazon.com/AmazonRDS/latest/UserGuide/CHAP_SQLServer.html#SQLServer.Concepts.General.TimeZone) for more information

~> **NOTE:** Removing the `replicate_source_db` attribute from an existing RDS
Replicate database managed by Terraform will promote the database to a fully
standalone database.

## Attributes Reference

The following attributes are exported:

* `id` - The RDS instance ID.
* `resource_id` - The RDS Resource ID of this instance.
* `address` - The address of the RDS instance.
* `arn` - The ARN of the RDS instance.
* `allocated_storage` - The amount of allocated storage
* `availability_zone` - The availability zone of the instance
* `backup_retention_period` - The backup retention period
* `backup_window` - The backup window
* `endpoint` - The connection endpoint
* `engine` - The database engine
* `engine_version` - The database engine version
* `instance_class`- The RDS instance class
* `maintenance_window` - The instance maintenance window
* `multi_az` - If the RDS instance is multi AZ enabled
* `name` - The database name
* `port` - The database port
* `status` - The RDS instance status
* `username` - The master username for the database
* `storage_encrypted` - Specifies whether the DB instance is encrypted
* `hosted_zone_id` - The canonical hosted zone ID of the DB instance (to be used in a Route 53 Alias record)

On Oracle instances the following is exported additionally:

* `character_set_name` - The character set used on Oracle instances.


<a id="timeouts"></a>
## Timeouts

`aws_db_instance` provides the following
[Timeouts](/docs/configuration/resources.html#timeouts) configuration options:

- `create` - (Default `40 minutes`) Used for Creating Instances, Replicas, and
restoring from Snapshots
- `update` - (Default `80 minutes`) Used for Database modifications
- `delete` - (Default `40 minutes`) Used for destroying databases. This includes
the time required to take snapshots

[1]: https://docs.aws.amazon.com/AmazonRDS/latest/UserGuide/Overview.Replication.html
[2]: https://docs.aws.amazon.com/AmazonRDS/latest/UserGuide/USER_UpgradeDBInstance.Maintenance.html

## Import

DB Instances can be imported using the `identifier`, e.g.

```
$ terraform import aws_db_instance.default mydb-rds-instance
```
