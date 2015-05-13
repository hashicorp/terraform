---
layout: "aws"
page_title: "AWS: aws_subnet"
sidebar_current: "docs-aws-resource-elasticache-cluster"
description: |-
  Provides an VPC subnet resource.
---

# aws\_elasticache\_cluster

Provides an ElastiCache Cluster resource. 

## Example Usage

```
resource "aws_elasticache_cluster" "bar" {
    cluster_id = "cluster-example"
    engine = "memcached"
    node_type = "cache.m1.small"
    num_cache_nodes = 1
    parameter_group_name = "default.memcached1.4"
}
```

## Argument Reference

The following arguments are supported:

* `cluster_id` – (Required) Group identifier. This parameter is stored as a 
lowercase string

* `engine` – (Required) Name of the cache engine to be used for this cache cluster.
 Valid values for this parameter are `memcached` or `redis`

* `engine_version` – (Optional) Version number of the cache engine to be used.
See [Selecting a Cache Engine and Version](http://docs.aws.amazon.com/AmazonElastiCache/latest/UserGuide/SelectEngine.html) 
in the AWS Documentation center for supported versions 

* `node_type` – (Required) The compute and memory capacity of the nodes. See 
[Available Cache Node Types](http://aws.amazon.com/elasticache/details#Available_Cache_Node_Types) for
supported node types

* `num_cache_nodes` – (Required) The initial number of cache nodes that the 
cache cluster will have. For Redis, this value must be 1. For Memcache, this
value must be between 1 and 20

* `parameter_group_name` – (Optional) Name of the parameter group to associate 
with this cache cluster

* `port` – (Optional) The port number on which each of the cache nodes will 
accept connections. Default 11211.

* `subnet_group_name` – (Optional, VPC only) Name of the subnet group to be used 
for the cache cluster.

* `security_group_names` – (Optional, EC2 Classic only) List of security group 
names to associate with this cache cluster

* `security_group_ids` – (Optional, VPC only) One or more VPC security groups associated 
 with the cache cluster


## Attributes Reference

The following attributes are exported:

* `cluster_id` 
* `engine` 
* `engine_version`
* `node_type`
* `num_cache_nodes`
* `parameter_group_name`
* `port` 
* `subnet_group_name`
* `security_group_names`
* `security_group_ids`
