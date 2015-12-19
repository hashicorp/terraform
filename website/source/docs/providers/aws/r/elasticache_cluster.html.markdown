---
layout: "aws"
page_title: "AWS: aws_elasticache_cluster"
sidebar_current: "docs-aws-resource-elasticache-cluster"
description: |-
  Provides an ElastiCache Cluster resource.
---

# aws\_elasticache\_cluster

Provides an ElastiCache Cluster resource.

Changes to a Cache Cluster can occur when you manually change a
parameter, such as `node_type`, and are reflected in the next maintenance
window. Because of this, Terraform may report a difference in it's planning
phase because a modification has not yet taken place. You can use the
`apply_immediately` flag to instruct the service to apply the change immediately 
(see documentation below). 

~> **Note:** using `apply_immediately` can result in a 
brief downtime as the server reboots. See the AWS Docs on 
[Modifying an ElastiCache Cache Cluster][2] for more information.

## Example Usage

```
resource "aws_elasticache_cluster" "bar" {
    cluster_id = "cluster-example"
    engine = "memcached"
    node_type = "cache.m1.small"
    port = 11211
    num_cache_nodes = 1
    parameter_group_name = "default.memcached1.4"
}
```

## Argument Reference

The following arguments are supported:

* `cluster_id` – (Required) Group identifier. Elasticache converts
  this name to lowercase

* `engine` – (Required) Name of the cache engine to be used for this cache cluster.
 Valid values for this parameter are `memcached` or `redis`

* `engine_version` – (Optional) Version number of the cache engine to be used.
See [Selecting a Cache Engine and Version](http://docs.aws.amazon.com/AmazonElastiCache/latest/UserGuide/SelectEngine.html)
in the AWS Documentation center for supported versions

* `maintenance_window` – (Optional) Specifies the weekly time range which maintenance 
on the cache cluster is performed. The format is `ddd:hh24:mi-ddd:hh24:mi` (24H Clock UTC). 
The minimum maintenance window is a 60 minute period. Example: `sun:05:00-sun:09:00`

* `node_type` – (Required) The compute and memory capacity of the nodes. See
[Available Cache Node Types](http://aws.amazon.com/elasticache/details#Available_Cache_Node_Types) for
supported node types

* `num_cache_nodes` – (Required) The initial number of cache nodes that the
cache cluster will have. For Redis, this value must be 1. For Memcache, this
value must be between 1 and 20. If this number is reduced on subsequent runs,
the highest numbered nodes will be removed.

* `parameter_group_name` – (Required) Name of the parameter group to associate
with this cache cluster

* `port` – (Required) The port number on which each of the cache nodes will
accept connections. For Memcache the default is 11211, and for Redis the default port is 6379.

* `subnet_group_name` – (Optional, VPC only) Name of the subnet group to be used
for the cache cluster.

* `security_group_names` – (Optional, EC2 Classic only) List of security group
names to associate with this cache cluster

* `security_group_ids` – (Optional, VPC only) One or more VPC security groups associated
 with the cache cluster

* `apply_immediately` - (Optional) Specifies whether any database modifications
     are applied immediately, or during the next maintenance window. Default is
     `false`. See [Amazon ElastiCache Documentation for more information.][1]
     (Available since v0.6.0)

* `snapshot_arns` – (Optional) A single-element string list containing an 
Amazon Resource Name (ARN) of a Redis RDB snapshot file stored in Amazon S3. 
Example: `arn:aws:s3:::my_bucket/snapshot1.rdb`

* `snapshot_window` - (Optional) The daily time range (in UTC) during which ElastiCache will 
begin taking a daily snapshot of your cache cluster. Can only be used for the Redis engine. Example: 05:00-09:00

* `snapshot_retention_limit` - (Optional) The number of days for which ElastiCache will 
retain automatic cache cluster snapshots before deleting them. For example, if you set 
SnapshotRetentionLimit to 5, then a snapshot that was taken today will be retained for 5 days 
before being deleted. If the value of SnapshotRetentionLimit is set to zero (0), backups are turned off. 
Can only be used for the Redis engine.

* `notification_topic_arn` – (Optional) An Amazon Resource Name (ARN) of an 
SNS topic to send ElastiCache notifications to. Example: 
`arn:aws:sns:us-east-1:012345678999:my_sns_topic`

* `tags` - (Optional) A mapping of tags to assign to the resource.

~> **NOTE:** Snapshotting functionality is not compatible with t2 instance types.

## Attributes Reference

The following attributes are exported:

* `cache_nodes` - List of node objects including `id`, `address` and `port`.
   Referenceable e.g. as `${aws_elasticache_cluster.bar.cache_nodes.0.address}`
   
* `configuration_endpoint` - (Memcached only) The configuration endpoint to allow host discovery

[1]: http://docs.aws.amazon.com/AmazonElastiCache/latest/APIReference/API_ModifyCacheCluster.html
[2]: http://docs.aws.amazon.com/fr_fr/AmazonElastiCache/latest/UserGuide/Clusters.Modify.html
