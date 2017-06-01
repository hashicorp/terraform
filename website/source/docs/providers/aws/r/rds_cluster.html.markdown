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

Changes to a RDS Cluster can occur when you manually change a
parameter, such as `port`, and are reflected in the next maintenance
window. Because of this, Terraform may report a difference in its planning
phase because a modification has not yet taken place. You can use the
`apply_immediately` flag to instruct the service to apply the change immediately
(see documentation below).

~> **Note:** using `apply_immediately` can result in a
brief downtime as the server reboots. See the AWS Docs on [RDS Maintenance][4]
for more information.

~> **Note:** All arguments including the username and password will be stored in the raw state as plain-text.
[Read more about sensitive data in state](/docs/state/sensitive-data.html).

## Example Usage

```hcl
resource "aws_rds_cluster" "default" {
  cluster_identifier      = "aurora-cluster-demo"
  availability_zones      = ["us-west-2a", "us-west-2b", "us-west-2c"]
  database_name           = "mydb"
  master_username         = "foo"
  master_password         = "bar"
  backup_retention_period = 5
  preferred_backup_window = "07:00-09:00"
}
```

~> **NOTE:** RDS Clusters resources that are created without any matching
RDS Cluster Instances do not currently display in the AWS Console.

## Argument Reference

For more detailed documentation about each argument, refer to
the [AWS official documentation](https://docs.aws.amazon.com/AmazonRDS/latest/CommandLineReference/CLIReference-cmd-ModifyDBInstance.html).

The following arguments are supported:

* `cluster_identifier` - (Optional, Forces new resources) The cluster identifier. If omitted, Terraform will assign a random, unique identifier.
* `cluster_identifier_prefix` - (Optional, Forces new resource) Creates a unique cluster identifier beginning with the specified prefix. Conflicts with `cluster_identifer`.
* `database_name` - (Optional) The name for your database of up to 8 alpha-numeric
  characters. If you do not provide a name, Amazon RDS will not create a
  database in the DB cluster you are creating
* `master_password` - (Required unless a `snapshot_identifier` is provided) Password for the master DB user. Note that this may
    show up in logs, and it will be stored in the state file
* `master_username` - (Required unless a `snapshot_identifier` is provided) Username for the master DB user
* `final_snapshot_identifier` - (Optional) The name of your final DB snapshot
    when this DB cluster is deleted. If omitted, no final snapshot will be
    made.
* `skip_final_snapshot` - (Optional) Determines whether a final DB snapshot is created before the DB cluster is deleted. If true is specified, no DB snapshot is created. If false is specified, a DB snapshot is created before the DB cluster is deleted, using the value from `final_snapshot_identifier`. Default is `false`.
* `availability_zones` - (Optional) A list of EC2 Availability Zones that
  instances in the DB cluster can be created in
* `backup_retention_period` - (Optional) The days to retain backups for. Default
1
* `preferred_backup_window` - (Optional) The daily time range during which automated backups are created if automated backups are enabled using the BackupRetentionPeriod parameter.Time in UTC
Default: A 30-minute window selected at random from an 8-hour block of time per region. e.g. 04:00-09:00
* `preferred_maintenance_window` - (Optional) The weekly time range during which system maintenance can occur, in (UTC) e.g. wed:04:00-wed:04:30
* `port` - (Optional) The port on which the DB accepts connections
* `vpc_security_group_ids` - (Optional) List of VPC security groups to associate
  with the Cluster
* `snapshot_identifier` - (Optional) Specifies whether or not to create this cluster from a snapshot. This correlates to the snapshot ID you'd find in the RDS console, e.g: rds:production-2015-06-26-06-05.
* `storage_encrypted` - (Optional) Specifies whether the DB cluster is encrypted. The default is `false` if not specified.
* `apply_immediately` - (Optional) Specifies whether any cluster modifications
     are applied immediately, or during the next maintenance window. Default is
     `false`. See [Amazon RDS Documentation for more information.](https://docs.aws.amazon.com/AmazonRDS/latest/UserGuide/Overview.DBInstance.Modifying.html)
* `db_subnet_group_name` - (Optional) A DB subnet group to associate with this DB instance. **NOTE:** This must match the `db_subnet_group_name` specified on every [`aws_rds_cluster_instance`](/docs/providers/aws/r/rds_cluster_instance.html) in the cluster.
* `db_cluster_parameter_group_name` - (Optional) A cluster parameter group to associate with the cluster.
* `kms_key_id` - (Optional) The ARN for the KMS encryption key. When specifying `kms_key_id`, `storage_encrypted` needs to be set to true.
* `iam_database_authentication_enabled` - (Optional) Specifies whether or mappings of AWS Identity and Access Management (IAM) accounts to database accounts is enabled.

## Attributes Reference

The following attributes are exported:

* `id` - The RDS Cluster Identifier
* `cluster_identifier` - The RDS Cluster Identifier
* `cluster_resource_id` - The RDS Cluster Resource ID
* `cluster_members` – List of RDS Instances that are a part of this cluster
* `allocated_storage` - The amount of allocated storage
* `availability_zones` - The availability zone of the instance
* `backup_retention_period` - The backup retention period
* `preferred_backup_window` - The backup window
* `preferred_maintenance_window` - The maintenance window
* `endpoint` - The DNS address of the RDS instance
* `reader_endpoint` - A read-only endpoint for the Aurora cluster, automatically
load-balanced across replicas
* `engine` - The database engine
* `engine_version` - The database engine version
* `maintenance_window` - The instance maintenance window
* `database_name` - The database name
* `port` - The database port
* `status` - The RDS instance status
* `master_username` - The master username for the database
* `storage_encrypted` - Specifies whether the DB cluster is encrypted
* `preferred_backup_window` - The daily time range during which the backups happen
* `replication_source_identifier` - ARN  of the source DB cluster if this DB cluster is created as a Read Replica.

[1]: https://docs.aws.amazon.com/AmazonRDS/latest/UserGuide/Overview.Replication.html
[2]: https://docs.aws.amazon.com/AmazonRDS/latest/UserGuide/CHAP_Aurora.html
[3]: /docs/providers/aws/r/rds_cluster_instance.html
[4]: https://docs.aws.amazon.com/AmazonRDS/latest/UserGuide/USER_UpgradeDBInstance.Maintenance.html

## Timeouts

`aws_rds_cluster` provides the following
[Timeouts](/docs/configuration/resources.html#timeouts) configuration options:

- `create` - (Default `120 minutes`) Used for Cluster creation
- `update` - (Default `120 minutes`) Used for Cluster modifications
- `delete` - (Default `120 minutes`) Used for destroying cluster. This includes
any cleanup task during the destroying process.

## Import

RDS Clusters can be imported using the `cluster_identifier`, e.g.

```
$ terraform import aws_rds_cluster.aurora_cluster aurora-prod-cluster
```
