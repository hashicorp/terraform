---
layout: "alicloud"
page_title: "Alicloud: alicloud_ess_scaling_group"
sidebar_current: "docs-alicloud-resource-ess-scaling-group"
description: |-
  Provides a ESS scaling group resource.
---

# alicloud\_ess\_scaling\_group

Provides a ESS scaling group resource.

## Example Usage

```
resource "alicloud_ess_scaling_group" "scaling" {
  min_size           = 1
  max_size           = 2
  removal_policies   = ["OldestInstance", "NewestInstance"]
}
```

## Argument Reference

The following arguments are supported:

* `min_size` - (Required) Minimum number of ECS instances in the scaling group. Value range: [0, 100].
* `max_size` - (Required) Maximum number of ECS instances in the scaling group. Value range: [0, 100].
* `scaling_group_name` - (Optional) Name shown for the scaling group, which must contain 2-40 characters (English or Chinese). If this parameter is not specified, the default value is ScalingGroupId.
* `default_cooldown` - (Optional) Default cool-down time (in seconds) of the scaling group. Value range: [0, 86400]. The default value is 300s.
* `vswitch_id` - (Optional) The virtual switch ID which the ecs instance to be create in.
* `removal_policies` - (Optional) RemovalPolicy is used to select the ECS instances you want to remove from the scaling group when multiple candidates for removal exist. Optional values:
    - OldestInstance: removes the first ECS instance attached to the scaling group.
    - NewestInstance: removes the first ECS instance attached to the scaling group.
    - OldestScalingConfiguration: removes the ECS instance with the oldest scaling configuration.
    - Default values: OldestScalingConfiguration and OldestInstance. You can enter up to two removal policies.
* `db_instance_ids` - (Optional) If an RDS instance is specified in the scaling group, the scaling group automatically attaches the Intranet IP addresses of its ECS instances to the RDS access whitelist.
    - The specified RDS instance must be in running status.
    - The specified RDS instance’s whitelist must have room for more IP addresses.
* `loadbalancer_ids` - (Optional) If a Server Load Balancer instance is specified in the scaling group, the scaling group automatically attaches its ECS instances to the Server Load Balancer instance.
    - The Server Load Balancer instance must be enabled.
    - Health check must be enabled for all listener ports configured for the Server Load Balancer instance; otherwise, creation fails.
    - The Server Load Balancer instance attached with VPC-type ECS instances cannot be attached to the scaling group.
    - The default weight of an ECS instance attached to the Server Load Balancer instance is 50.

## Attributes Reference

The following attributes are exported:

* `id` - The scaling group ID.
* `min_size` - The minimum number of ECS instances.
* `max_size` - The maximum number of ECS instances.
* `scaling_group_name` - The name of the scaling group.
* `default_cooldown` - The default cool-down of the scaling group.
* `removal_policies` - The removal policy used to select the ECS instance to remove from the scaling group.
* `db_instance_ids` - The db instance id which the ECS instance attached to.
* `loadbalancer_ids` - The slb instance id which the ECS instance attached to.