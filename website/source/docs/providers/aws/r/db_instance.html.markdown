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
* `backup_retention_period` - (Optional) The days to retain backups for.
* `backup_window` - (Optional) The backup window.
* `iops` - (Optional) The amount of provisioned IOPS. Setting this implies a
    storage_type of "io1".
* `maintenance_window` - (Optional) The window to perform maintenance in.
* `multi_az` - (Optional) Specifies if the RDS instance is multi-AZ
* `port` - (Optional) The port on which the DB accepts connections.
* `publicly_accessible` - (Optional) Bool to control if instance is publicly accessible.
* `vpc_security_group_ids` - (Optional) List of VPC security groups to associate.
* `security_group_names` - (Optional/Deprecated) List of DB Security Groups to associate.
    Only used for [DB Instances on the _EC2-Classic_ Platform](http://docs.aws.amazon.com/AmazonRDS/latest/UserGuide/USER_VPC.html#USER_VPC.FindDefaultVPC). 
* `db_subnet_group_name` - (Optional) Name of DB subnet group
* `parameter_group_name` - (Optional) Name of the DB parameter group to associate.
* `storage_encrypted` - (Optional) Specifies whether the DB instance is encrypted. The Default is `false` if not specified.

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

