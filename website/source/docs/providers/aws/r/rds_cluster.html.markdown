---
layout: "aws"
page_title: "AWS: aws_rds_cluster"
sidebar_current: "docs-aws-resource-rds-cluster"
description: |-
  Provides an RDS Cluster Resource
---

# aws\_rds\_cluster

Provides an RDS Cluster Resource. A Cluster Resource defines attributes that are
applied to the entire cluster of [RDS Cluster Instances][3]. Use the RDS Cluster
resource and RDS Cluster Instances to create and use Amazon Aurora, a MySQL-compatible
database engine.

For more information on Amazon Aurora, see [Aurora on Amazon RDS][2] in the Amazon RDS User Guide.

## Example Usage

```
resource "aws_rds_cluster" "default" {
  cluster_identifier = "aurora-cluster-demo"
  availability_zones = ["us-west-2a","us-west-2b","us-west-2c"]
  database_name = "mydb"
  master_username = "foo"
  master_password = "bar"
}
```

~> **NOTE:** RDS Clusters resources that are created without any matching
RDS Cluster Instances do not currently display in the AWS Console.

## Argument Reference

For more detailed documentation about each argument, refer to
the [AWS official documentation](http://docs.aws.amazon.com/AmazonRDS/latest/CommandLineReference/CLIReference-cmd-ModifyDBInstance.html).

The following arguments are supported:

* `cluster_identifier` - (Required) The Cluster Identifier. Must be a lower case
string.
* `database_name` - (Optional) The name for your database of up to 8 alpha-numeric
  characters. If you do not provide a name, Amazon RDS will not create a
  database in the DB cluster you are creating
* `master_password` - (Required) Password for the master DB user. Note that this may
    show up in logs, and it will be stored in the state file
* `master_username` - (Required) Username for the master DB user
* `final_snapshot_identifier` - (Optional) The name of your final DB snapshot
    when this DB cluster is deleted. If omitted, no final snapshot will be
    made.
* `availability_zones` - (Optional) A list of EC2 Availability Zones that
  instances in the DB cluster can be created in
* `backup_retention_period` - (Optional) The days to retain backups for. Default
1
* `port` - (Optional) The port on which the DB accepts connections
* `vpc_security_group_ids` - (Optional) List of VPC security groups to associate
  with the Cluster
* `apply_immediately` - (Optional) Specifies whether any cluster modifications
     are applied immediately, or during the next maintenance window. Default is
     `false`. See [Amazon RDS Documentation for more information.](http://docs.aws.amazon.com/AmazonRDS/latest/UserGuide/Overview.DBInstance.Modifying.html)

## Attributes Reference

The following attributes are exported:

* `id` - The RDS Cluster Identifier
* `cluster_identifier` - The RDS Cluster Identifier
* `cluster_members` – List of RDS Instances that are a part of this cluster
* `address` - The address of the RDS instance.
* `allocated_storage` - The amount of allocated storage
* `availability_zones` - The availability zone of the instance
* `backup_retention_period` - The backup retention period
* `backup_window` - The backup window
* `endpoint` - The primary, writeable connection endpoint
* `engine` - The database engine
* `engine_version` - The database engine version
* `maintenance_window` - The instance maintenance window
* `database_name` - The database name
* `port` - The database port
* `status` - The RDS instance status
* `username` - The master username for the database
* `storage_encrypted` - Specifies whether the DB instance is encrypted

[1]: http://docs.aws.amazon.com/AmazonRDS/latest/UserGuide/Overview.Replication.html

[2]: http://docs.aws.amazon.com/AmazonRDS/latest/UserGuide/CHAP_Aurora.html
[3]: /docs/providers/aws/r/rds_cluster_instance.html
