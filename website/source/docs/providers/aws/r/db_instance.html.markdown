---
layout: "aws"
page_title: "AWS: aws_db_instance"
sidebar_current: "docs-aws-resource-db-instance"
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
	security_group_names = ["${aws_db_security_group.bar.name}"]
        subnet_group_name = "my_database_subnet_group"
}
```

## Argument Reference

The following arguments are supported:

* `allocated_storage` - (Required) The allocated storage in gigabytes.
* `engine` - (Required) The database engine to use.
* `engine_version` - (Required) The engine version to use.
* `identifier` - (Required) The name of the RDS instance
* `instance_class` - (Required) The instance type of the RDS instance.
* `final_snapshot_identifier` - (Optional) The name of your final DB snapshot.
* `name` - (Required) The DB name to create.
* `password` - (Required) Password for the master DB user. Note that this will be stored
    in the state file.
* `username` - (Required) Username for the master DB user.
* `availability_zone` - (Optional) The AZ for the RDS instance.
* `backup_retention_period` - (Optional) The days to retain backups for.
* `backup_window` - (Optional) The backup window.
* `iops` - (Optional) The amount of provisioned IOPS
* `maintenance_window` - (Optional) The window to perform maintenance in.
* `multi_az` - (Optional) Specifies if the RDS instance is multi-AZ
* `port` - (Optional) The port on which the DB accepts connections.
* `publicly_accessible` - (Optional) Bool to control if instance is publicly accessible.
* `vpc_security_group_ids` - (Optional) List of VPC security groups to associate.
* `skip_final_snapshot` - (Optional) Enables skipping the final snapshot on deletion.
* `security_group_names` - (Optional) List of DB Security Groups to associate.
* `subnet_group_name` - (Optional) Name of DB subnet group

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

