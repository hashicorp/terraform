---
layout: "aws"
page_title: "AWS: aws_elasticache_replication_group"
sidebar_current: "docs-aws-resource-elasticache-replication-group"
description: |-
  Provides an ElastiCache Replication Group resource.
---

# aws\_elasticache\_replication\_group

Provides an ElastiCache Replication Group resource.

## Example Usage

```
resource "aws_elasticache_replication_group" "redis" {
    replication_group_id = "users-redis"
    description = "users redis"
    engine = "redis"
    cache_node_type = "cache.m3.medium"
    num_cache_clusters = 2
    automatic_failover = true
    subnet_group_name = "${aws_elasticache_subnet_group.redis.name}"
    security_group_ids = ["${aws_security_group.redis.id}"]
}
```

## Argument Reference

The following arguments are supported:

* `replication_group_id` – (Required) Replication group identifier. This
parameter is stored as a lowercase string

* `description` – (Required) The description of the replication group.

* `engine` – (Optional) The name of the cache engine to be used for the cache clusters in this replication group.
 The only current valid value is `redis`

* `engine_version` – (Optional) Version number of the cache engine to be used.
See [Selecting a Cache Engine and Version](http://docs.aws.amazon.com/AmazonElastiCache/latest/UserGuide/SelectEngine.html)
in the AWS Documentation center for supported versions

* `cache_node_type` – (Required) The compute and memory capacity of the nodes. See
[Available Cache Node Types](http://aws.amazon.com/elasticache/details#Available_Cache_Node_Types) for
supported node types

* `automatic_failover` - (Optional) Specifies whether a read-only replica will be automatically promoted to read/write primary if the existing primary fails.
If true, Multi-AZ is enabled for this replication group. If false, Multi-AZ is disabled for this replication group.

* `num_cache_clusters` – (Optional) The number of cache clusters this replication group will initially have. If `automatic_failover` is enabled, the value of this parameter must be at least 2.
Either this or `primary_cluster_id` is required.

* `primary_cluster_id` - (Optional) The identifier of the cache cluster that
will serve as the primary for this replication group. This cache cluster must already exist and have a status of available.
Either this or `num_cache_clusters` is required.

* `parameter_group_name` – (Required) Name of the parameter group to associate
with this cache cluster

* `preferred_cache_cluster_azs` - (Optional) A list of EC2 availability zones in which the replication group's cache clusters will be created. The order of the availability zones in the list is not important. If not provided, AWS will chose them for you.

* `subnet_group_name` – (Optional, VPC only) Name of the subnet group to be used
for the cache cluster.

* `security_group_names` – (Optional, EC2 Classic only) List of security group
names to associate with this cache cluster

* `security_group_ids` – (Optional, VPC only) One or more VPC security groups associated
 with the cache cluster


## Attributes Reference

The following attributes are exported:

* `primary_endpoint` - The address of the primary node.
