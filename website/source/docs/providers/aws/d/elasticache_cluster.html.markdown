---
layout: "aws"
page_title: "AWS: aws_elasticache_cluster"
sidebar_current: "docs-aws-datasource-elasticache-cluster"
description: |-
  Get information on an ElastiCache Cluster resource.
---

# aws_elasticache_cluster

Use this data source to get information about an Elasticache Cluster

## Example Usage

```hcl
data "aws_elasticache_cluster" "my_cluster" {
  cluster_id = "my-cluster-id"
}

## Argument Reference

The following arguments are supported:

* `cluster_id` – (Required) Group identifier.


## Attributes Reference

The following attributes are exported:

* `node_type` – The cluster node type.
* `num_cache_nodes` – The number of cache nodes that the cache cluster has.
* `engine` – Name of the cache engine.
* `engine_version` – Version number of the cache engine.
* `subnet_group_name` – Name of the subnet group associated to the cache cluster.
* `security_group_names` – List of security group names associated with this cache cluster.
* `security_group_ids` – List VPC security groups associated with the cache cluster.
* `parameter_group_name` – Name of the parameter group associated with this cache cluster.
* `replication_group_id` - The replication group to which this cache cluster belongs.
* `maintenance_window` – Specifies the weekly time range for when maintenance
on the cache cluster is performed.
* `snapshot_window` - The daily time range (in UTC) during which ElastiCache will
begin taking a daily snapshot of the cache cluster.
* `snapshot_retention_limit` - The number of days for which ElastiCache will
retain automatic cache cluster snapshots before deleting them.
* `availability_zone` - The Availability Zone for the cache cluster.
* `notification_topic_arn` – An Amazon Resource Name (ARN) of an
SNS topic that ElastiCache notifications get sent to.
* `port` – The port number on which each of the cache nodes will
accept connections.
* `configuration_endpoint` - The configuration endpoint to allow host discovery.
* `cluster_address` - The DNS name of the cache cluster without the port appended.
* `cache_nodes` - List of node objects including `id`, `address`, `port` and `availability_zone`.
   Referenceable e.g. as `${data.aws_elasticache_cluster.bar.cache_nodes.0.address}`
* `tags` - The tags assigned to the resource
