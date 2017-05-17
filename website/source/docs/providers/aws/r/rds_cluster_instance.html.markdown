---
layout: "aws"
page_title: "AWS: aws_rds_cluster_instance"
sidebar_current: "docs-aws-resource-rds-cluster-instance"
description: |-
  Provides an RDS Cluster Resource Instance
---

# aws\_rds\_cluster\_instance

Provides an RDS Cluster Resource Instance. A Cluster Instance Resource defines
attributes that are specific to a single instance in a [RDS Cluster][3],
specifically running Amazon Aurora.

Unlike other RDS resources that support replication, with Amazon Aurora you do
not designate a primary and subsequent replicas. Instead, you simply add RDS
Instances and Aurora manages the replication. You can use the [count][5]
meta-parameter to make multiple instances and join them all to the same RDS
Cluster, or you may specify different Cluster Instance resources with various
`instance_class` sizes.

For more information on Amazon Aurora, see [Aurora on Amazon RDS][2] in the Amazon RDS User Guide.

## Example Usage

```hcl
resource "aws_rds_cluster_instance" "cluster_instances" {
  count              = 2
  identifier         = "aurora-cluster-demo-${count.index}"
  cluster_identifier = "${aws_rds_cluster.default.id}"
  instance_class     = "db.r3.large"
}

resource "aws_rds_cluster" "default" {
  cluster_identifier = "aurora-cluster-demo"
  availability_zones = ["us-west-2a", "us-west-2b", "us-west-2c"]
  database_name      = "mydb"
  master_username    = "foo"
  master_password    = "barbut8chars"
}
```

## Argument Reference

For more detailed documentation about each argument, refer to
the [AWS official documentation](https://docs.aws.amazon.com/AmazonRDS/latest/CommandLineReference/CLIReference-cmd-ModifyDBInstance.html).

The following arguments are supported:

* `identifier` - (Optional, Forces new resource) The indentifier for the RDS instance, if omitted, Terraform will assign a random, unique identifier.
* `identifier_prefix` - (Optional, Forces new resource) Creates a unique identifier beginning with the specified prefix. Conflicts with `identifer`.
* `cluster_identifier` - (Required) The identifier of the [`aws_rds_cluster`](/docs/providers/aws/r/rds_cluster.html) in which to launch this instance.
* `instance_class` - (Required) The instance class to use. For details on CPU
and memory, see [Scaling Aurora DB Instances][4]. Aurora currently
  supports the below instance classes.
  - db.t2.small
  - db.t2.medium
  - db.r3.large
  - db.r3.xlarge
  - db.r3.2xlarge
  - db.r3.4xlarge
  - db.r3.8xlarge
* `publicly_accessible` - (Optional) Bool to control if instance is publicly accessible.
Default `false`. See the documentation on [Creating DB Instances][6] for more
details on controlling this property.
* `db_subnet_group_name` - (Required if `publicly_accessible = false`, Optional otherwise) A DB subnet group to associate with this DB instance. **NOTE:** This must match the `db_subnet_group_name` of the attached [`aws_rds_cluster`](/docs/providers/aws/r/rds_cluster.html).
* `db_parameter_group_name` - (Optional) The name of the DB parameter group to associate with this instance.
* `apply_immediately` - (Optional) Specifies whether any database modifications
     are applied immediately, or during the next maintenance window. Default is`false`.
* `monitoring_role_arn` - (Optional) The ARN for the IAM role that permits RDS to send
enhanced monitoring metrics to CloudWatch Logs. You can find more information on the [AWS Documentation](http://docs.aws.amazon.com/AmazonRDS/latest/UserGuide/USER_Monitoring.html)
what IAM permissions are needed to allow Enhanced Monitoring for RDS Instances.
* `monitoring_interval` - (Optional) The interval, in seconds, between points when Enhanced Monitoring metrics are collected for the DB instance. To disable collecting Enhanced Monitoring metrics, specify 0. The default is 0. Valid Values: 0, 1, 5, 10, 15, 30, 60.
* `promotion_tier` - (Optional) Default 0. Failover Priority setting on instance level. The reader who has lower tier has higher priority to get promoter to writer. 
* `preferred_backup_window` - (Optional) The daily time range during which automated backups are created if automated backups are enabled.
  Eg: "04:00-09:00"
* `preferred_maintenance_window` - (Optional) The window to perform maintenance in.
  Syntax: "ddd:hh24:mi-ddd:hh24:mi". Eg: "Mon:00:00-Mon:03:00".
* `auto_minor_version_upgrade` - (Optional) Indicates that minor engine upgrades will be applied automatically to the DB instance during the maintenance window. Default `true`.
* `tags` - (Optional) A mapping of tags to assign to the instance.

## Attributes Reference

The following attributes are exported:

* `cluster_identifier` - The RDS Cluster Identifier
* `identifier` - The Instance identifier
* `id` - The Instance identifier
* `writer` – Boolean indicating if this instance is writable. `False` indicates
this instance is a read replica
* `allocated_storage` - The amount of allocated storage
* `availability_zones` - The availability zone of the instance
* `endpoint` - The DNS address for this instance. May not be writable
* `engine` - The database engine
* `engine_version` - The database engine version
* `database_name` - The database name
* `port` - The database port
* `status` - The RDS instance status
* `storage_encrypted` - Specifies whether the DB cluster is encrypted.
* `kms_key_id` - The ARN for the KMS encryption key if one is set to the cluster.

[2]: https://docs.aws.amazon.com/AmazonRDS/latest/UserGuide/CHAP_Aurora.html
[3]: /docs/providers/aws/r/rds_cluster.html
[4]: https://docs.aws.amazon.com/AmazonRDS/latest/UserGuide/Aurora.Managing.html
[5]: /docs/configuration/resources.html#count
[6]: https://docs.aws.amazon.com/AmazonRDS/latest/APIReference/API_CreateDBInstance.html

## Timeouts

`aws_rds_cluster_instance` provides the following
[Timeouts](/docs/configuration/resources.html#timeouts) configuration options:

- `create` - (Default `90 minutes`) Used for Creating Instances, Replicas, and
restoring from Snapshots
- `update` - (Default `90 minutes`) Used for Database modifications
- `delete` - (Default `90 minutes`) Used for destroying databases. This includes
the time required to take snapshots

## Import

RDS Cluster Instances can be imported using the `identifier`, e.g.

```
$ terraform import aws_rds_cluster_instance.prod_instance_1 aurora-cluster-instance-1
```
