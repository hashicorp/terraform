---
layout: "aws"
page_title: "AWS: aws_opsworks_haproxy_layer"
sidebar_current: "docs-aws-resource-opsworks-haproxy-layer"
description: |-
  Provides an OpsWorks HAProxy layer resource.
---

# aws\_opsworks\_haproxy\_layer

Provides an OpsWorks haproxy layer resource.

## Example Usage

```hcl
resource "aws_opsworks_haproxy_layer" "lb" {
  stack_id       = "${aws_opsworks_stack.main.id}"
  stats_password = "foobarbaz"
}
```

## Argument Reference

The following arguments are supported:

* `stack_id` - (Required) The id of the stack the layer will belong to.
* `stats_password` - (Required) The password to use for HAProxy stats.
* `name` - (Optional) A human-readable name for the layer.
* `auto_assign_elastic_ips` - (Optional) Whether to automatically assign an elastic IP address to the layer's instances.
* `auto_assign_public_ips` - (Optional) For stacks belonging to a VPC, whether to automatically assign a public IP address to each of the layer's instances.
* `custom_instance_profile_arn` - (Optional) The ARN of an IAM profile that will be used for the layer's instances.
* `custom_security_group_ids` - (Optional) Ids for a set of security groups to apply to the layer's instances.
* `auto_healing` - (Optional) Whether to enable auto-healing for the layer.
* `healthcheck_method` - (Optional) HTTP method to use for instance healthchecks. Defaults to "OPTIONS".
* `healthcheck_url` - (Optional) URL path to use for instance healthchecks. Defaults to "/".
* `install_updates_on_boot` - (Optional) Whether to install OS and package updates on each instance when it boots.
* `instance_shutdown_timeout` - (Optional) The time, in seconds, that OpsWorks will wait for Chef to complete after triggering the Shutdown event.
* `elastic_load_balancer` - (Optional) Name of an Elastic Load Balancer to attach to this layer
* `drain_elb_on_shutdown` - (Optional) Whether to enable Elastic Load Balancing connection draining.
* `stats_enabled` - (Optional) Whether to enable HAProxy stats.
* `stats_url` - (Optional) The HAProxy stats URL. Defaults to "/haproxy?stats".
* `stats_user` - (Optional) The username for HAProxy stats. Defaults to "opsworks".
* `system_packages` - (Optional) Names of a set of system packages to install on the layer's instances.
* `use_ebs_optimized_instances` - (Optional) Whether to use EBS-optimized instances.
* `ebs_volume` - (Optional) `ebs_volume` blocks, as described below, will each create an EBS volume and connect it to the layer's instances.
* `custom_json` - (Optional) Custom JSON attributes to apply to the layer.

The following extra optional arguments, all lists of Chef recipe names, allow
custom Chef recipes to be applied to layer instances at the five different
lifecycle events, if custom cookbooks are enabled on the layer's stack:

* `custom_configure_recipes`
* `custom_deploy_recipes`
* `custom_setup_recipes`
* `custom_shutdown_recipes`
* `custom_undeploy_recipes`

An `ebs_volume` block supports the following arguments:

* `mount_point` - (Required) The path to mount the EBS volume on the layer's instances.
* `size` - (Required) The size of the volume in gigabytes.
* `number_of_disks` - (Required) The number of disks to use for the EBS volume.
* `raid_level` - (Required) The RAID level to use for the volume.
* `type` - (Optional) The type of volume to create. This may be `standard` (the default), `io1` or `gp2`.
* `iops` - (Optional) For PIOPS volumes, the IOPS per disk.

## Attributes Reference

The following attributes are exported:

* `id` - The id of the layer.
