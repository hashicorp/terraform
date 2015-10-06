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

```
resource "aws_rds_cluster_instance" "cluster_instances" {
  count = 2
  identifier = "aurora-cluster-demo"
  cluster_identifer = "${aws_rds_cluster.default.id}"
  instance_class = "db.r3.large"
}

resource "aws_rds_cluster" "default" {
  cluster_identifier = "aurora-cluster-demo"
  availability_zones = ["us-west-2a","us-west-2b","us-west-2c"]
  database_name = "mydb"
  master_username = "foo"
  master_password = "bar"
}
```

## Argument Reference

For more detailed documentation about each argument, refer to
the [AWS official documentation](http://docs.aws.amazon.com/AmazonRDS/latest/CommandLineReference/CLIReference-cmd-ModifyDBInstance.html).

The following arguments are supported:

* `identifier` - (Required) The Instance Identifier. Must be a lower case
string.
* `cluster_identifier` - (Required) The Cluster Identifier for this Instance to
join. Must be a lower case
string.
* `instance_class` - (Required) The instance class to use. For details on CPU
and memory, see [Scaling Aurora DB Instances][4]. Aurora currently
  supports the below instance classes.
  - db.r3.large
  - db.r3.xlarge
  - db.r3.2xlarge
  - db.r3.4xlarge
  - db.r3.8xlarge
* `publicly_accessible` - (Optional) Bool to control if instance is publicly accessible.
Default `false`. See the documentation on [Creating DB Instances][6] for more
details on controlling this property.

## Attributes Reference

The following attributes are exported:

* `cluster_identifier` - The RDS Cluster Identifier
* `identifier` - The Instance identifier
* `id` - The Instance identifier
* `writer` – Boolean indicating if this instance is writable. `False` indicates
this instance is a read replica
* `allocated_storage` - The amount of allocated storage
* `availability_zones` - The availability zone of the instance
* `endpoint` - The IP address for this instance. May not be writable
* `engine` - The database engine
* `engine_version` - The database engine version
* `database_name` - The database name
* `port` - The database port
* `status` - The RDS instance status

[2]: http://docs.aws.amazon.com/AmazonRDS/latest/UserGuide/CHAP_Aurora.html
[3]: /docs/providers/aws/r/rds_cluster.html
[4]: http://docs.aws.amazon.com/AmazonRDS/latest/UserGuide/Aurora.Managing.html
[5]: /docs/configuration/resources.html#count
[6]: http://docs.aws.amazon.com/fr_fr/AmazonRDS/latest/APIReference/API_CreateDBInstance.html
