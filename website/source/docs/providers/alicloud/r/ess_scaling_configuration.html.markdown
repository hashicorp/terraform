---
layout: "alicloud"
page_title: "Alicloud: alicloud_ess_scaling_configuration"
sidebar_current: "docs-alicloud-resource-ess-scaling-configuration"
description: |-
  Provides a ESS scaling configuration resource.
---

# alicloud\_ess\_scaling\_configuration

Provides a ESS scaling configuration resource.

## Example Usage

```
resource "alicloud_security_group" "classic" {
  # Other parameters...
}
resource "alicloud_ess_scaling_group" "scaling" {
  min_size           = 1
  max_size           = 2
  removal_policies   = ["OldestInstance", "NewestInstance"]
}

resource "alicloud_ess_scaling_configuration" "config" {
  scaling_group_id  = "${alicloud_ess_scaling_group.scaling.id}"

  image_id          = "ubuntu_140405_64_40G_cloudinit_20161115.vhd"
  instance_type     = "ecs.s2.large"
  security_group_id = "${alicloud_security_group.classic.id}"
}

```

## Argument Reference

The following arguments are supported:

* `scaling_group_id` - (Required) ID of the scaling group of a scaling configuration.
* `image_id` - (Required) ID of an image file, indicating the image resource selected when an instance is enabled.
* `instance_type` - (Required) Resource type of an ECS instance.
* `io_optimized` - (Required) Valid values are `none`, `optimized`, If `optimized`, the launched ECS instance will be I/O optimized.
* `security_group_id` - (Required) ID of the security group to which a newly created instance belongs.
* `scaling_configuration_name` - (Optional) Name shown for the scheduled task. If this parameter value is not specified, the default value is ScalingConfigurationId.
* `internet_charge_type` - (Optional) Network billing type, Values: PayByBandwidth or PayByTraffic. If this parameter value is not specified, the default value is PayByBandwidth.
* `internet_max_bandwidth_in` - (Optional) Maximum incoming bandwidth from the public network, measured in Mbps (Mega bit per second). The value range is [1,200].
* `internet_max_bandwidth_out` - (Optional) Maximum outgoing bandwidth from the public network, measured in Mbps (Mega bit per second). The value range for PayByBandwidth is [1,100].
* `system_disk_category` - (Optional) Category of the system disk. The parameter value options are cloud and ephemeral. 
* `data_disk` - (Optional) DataDisk mappings to attach to ecs instance. See [Block datadisk](#block-datadisk) below for details. 
* `instance_ids` - (Optional) ID of the ECS instance to be attached to the scaling group after it is enabled. You can input up to 20 IDs. 


## Block datadisk

The datadisk mapping supports the following:

* `size` - (Optional) Size of data disk, in GB. The value ranges from 5 to 2,000 for a cloud disk and from 5 to 1,024 for an ephemeral disk. A maximum of four values can be entered. 
* `category` - (Optional) Category of data disk. The parameter value options are cloud and ephemeral.
* `snapshot_id` - (Optional) Snapshot used for creating the data disk. If this parameter is specified, the size parameter is neglected, and the size of the created disk is the size of the snapshot. 
* `device` - (Optional) Attaching point of the data disk. If this parameter is empty, the ECS automatically assigns the attaching point when an ECS is created. The parameter value ranges from /dev/xvdb to /dev/xvdz. Restrictions on attaching ECS instances:
    - The attached ECS instance and the scaling group must be in the same region.
    - The attached ECS instance and the instance with active scaling configurations must be of the same type.
    - The attached ECS instance must in the running state.
    - The attached ECS instance has not been attached to other scaling groups.
    - The attached ECS instance supports Subscription and Pay-As-You-Go payment methods.
    - If the VswitchID is specified for a scaling group, you cannot attach Classic ECS instances or ECS instances on other VPCs to the scaling group.
    - If the VswitchID is not specified for the scaling group, ECS instances of the VPC type cannot be attached to the scaling group
* `active` - (Optional) If active current scaling configuration in the scaling group. 
* `enable` - (Optional) Enables the specified scaling group.
    - After the scaling group is successfully enabled (the group is active), the ECS instances specified by the interface are attached to the group.
    - If the current number of ECS instances in the scaling group is still smaller than MinSize after the ECS instances specified by the interface are attached, the Auto Scaling service automatically creates ECS instances in Pay-As-You-Go mode to make odds even. For example, a scaling group is created with MinSize = 5. Two existing ECS instances are specified by the InstanceId.N parameter when the scaling group is enabled. Three additional ECS instances are automatically created after the two ECS instances are attached by the Auto Scaling service to the scaling group.

## Attributes Reference

The following attributes are exported:

* `id` - The scaling configuration ID.
* `active` - Wether the current scaling configuration is actived.
* `image_id` - The ecs instance Image id.
* `instance_type` - The ecs instance type.
* `io_optimized` - The ecs instance whether I/O optimized.
* `security_group_id` - ID of the security group to which a newly created instance belongs.
* `scaling_configuration_name` - Name of scaling configuration.
* `internet_charge_type` - Internet charge type of ecs instance.