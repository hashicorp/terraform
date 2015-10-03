---
layout: "aws"
page_title: "AWS: aws_db_instance"
sidebar_current: "docs-aws-resource-db-instance"
description: |-
  Provides an RDS instance resource.
---

# aws\_db\_instance

Provides an RDS instance resource.

## Example Usage

```
resource "aws_db_instance" "default" {
	identifier = "mydb-rds"
	allocated_storage = 10
	engine = "mysql"
	engine_version = "5.6.17"
	instance_class = "db.t1.micro"
	name = "mydb"
	username = "foo"
	password = "bar"
	db_subnet_group_name = "my_database_subnet_group"
	parameter_group_name = "default.mysql5.6"
}
```

## Argument Reference

For more detailed documentation about each argument, refer to
the [AWS official documentation](http://docs.aws.amazon.com/AmazonRDS/latest/CommandLineReference/CLIReference-cmd-ModifyDBInstance.html).

The following arguments are supported:

* `allocated_storage` - (Required) The allocated storage in gigabytes.
* `engine` - (Required) The database engine to use.
* `engine_version` - (Required) The engine version to use.
* `identifier` - (Required) The name of the RDS instance
* `instance_class` - (Required) The instance type of the RDS instance.
* `storage_type` - (Optional) One of "standard" (magnetic), "gp2" (general
	purpose SSD), or "io1" (provisioned IOPS SSD). The default is "io1" if
	`iops` is specified, "standard" if not.
* `final_snapshot_identifier` - (Optional) The name of your final DB snapshot
    when this DB instance is deleted. If omitted, no final snapshot will be
    made.
* `name` - (Optional) The DB name to create. If omitted, no database is created
    initially.
* `password` - (Required) Password for the master DB user. Note that this may
    show up in logs, and it will be stored in the state file.
* `username` - (Required) Username for the master DB user.
* `availability_zone` - (Optional) The AZ for the RDS instance.
* `backup_retention_period` - (Optional) The days to retain backups for. Must be
`1` or greater to be a source for a [Read Replica][1].
* `backup_window` - (Optional) The backup window.
* `iops` - (Optional) The amount of provisioned IOPS. Setting this implies a
    storage_type of "io1".
* `maintenance_window` - (Optional) The window to perform maintenance in.
  Syntax: "ddd:hh24:mi-ddd:hh24:mi". Eg: "Mon:00:00-Mon:03:00".
  See [RDS Maintenance Window docs](http://docs.aws.amazon.com/AmazonRDS/latest/UserGuide/AdjustingTheMaintenanceWindow.html) for more.
* `multi_az` - (Optional) Specifies if the RDS instance is multi-AZ
* `port` - (Optional) The port on which the DB accepts connections.
* `publicly_accessible` - (Optional) Bool to control if instance is publicly accessible.
* `vpc_security_group_ids` - (Optional) List of VPC security groups to associate.
* `security_group_names` - (Optional/Deprecated) List of DB Security Groups to associate.
    Only used for [DB Instances on the _EC2-Classic_ Platform](http://docs.aws.amazon.com/AmazonRDS/latest/UserGuide/USER_VPC.html#USER_VPC.FindDefaultVPC).
* `db_subnet_group_name` - (Optional) Name of DB subnet group. DB instance will be created in the VPC associated with the DB subnet group. If unspecified, will be created in the `default` VPC, or in EC2 Classic, if available.
* `parameter_group_name` - (Optional) Name of the DB parameter group to associate.
* `storage_encrypted` - (Optional) Specifies whether the DB instance is encrypted. The default is `false` if not specified.
* `apply_immediately` - (Optional) Specifies whether any database modifications
     are applied immediately, or during the next maintenance window. Default is
     `false`. See [Amazon RDS Documentation for more information.](http://docs.aws.amazon.com/AmazonRDS/latest/UserGuide/Overview.DBInstance.Modifying.html)
* `replicate_source_db` - (Optional) Specifies that this resource is a Replicate
database, and to use this value as the source database. This correlates to the
`identifier` of another Amazon RDS Database to replicate. See
[DB Instance Replication][1] and
[Working with PostgreSQL and MySQL Read Replicas](http://docs.aws.amazon.com/AmazonRDS/latest/UserGuide/USER_ReadRepl.html) for
 more information on using Replication.
* `snapshot_identifier` - (Optional) Specifies whether or not to create this database from a snapshot. This correlates to the snapshot ID you'd find in the RDS console, e.g: rds:production-2015-06-26-06-05.

~> **NOTE:** Removing the `replicate_source_db` attribute from an existing RDS
Replicate database managed by Terraform will promote the database to a fully
standalone database.

## Attributes Reference

The following attributes are exported:

* `id` - The RDS instance ID.
* `address` - The address of the RDS instance.
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

[1]: http://docs.aws.amazon.com/AmazonRDS/latest/UserGuide/Overview.Replication.html
