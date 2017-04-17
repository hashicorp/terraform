---
layout: "aws"
page_title: "AWS: aws_emr_instance_group"
sidebar_current: "docs-aws-resource-emr-instance-group"
description: |-
  Provides an Elastic MapReduce Cluster Instance Group
---

# aws\_emr\_instance\_group

Provides an Elastic MapReduce Cluster Instance Group configuration.
See [Amazon Elastic MapReduce Documentation](https://aws.amazon.com/documentation/emr/) for more information.

~> **NOTE:** At this time, Instance Groups cannot be destroyed through the API nor
web interface. Instance Groups are destroyed when the EMR Cluster is destroyed.
Terraform will resize any Instance Group to zero when destroying the resource.

## Example Usage

```hcl
resource "aws_emr_instance_group" "task" {
  cluster_id     = "${aws_emr_cluster.tf-test-cluster.id}"
  instance_count = 1
  instance_type  = "m3.xlarge"
  name           = "my little instance group"
}
```

## Argument Reference

The following arguments are supported:
* `name` (Required) Human friendly name given to the instance group. Changing this forces a new resource to be created.
* `cluster_id` (Required) ID of the EMR Cluster to attach to. Changing this forces a new resource to be created.
* `instance_type` (Required) The EC2 instance type for all instances in the instance group. Changing this forces a new resource to be created.
* `instance_count` (Optional) Target number of instances for the instance group. Defaults to 0.
* `ebs_optimized` (Optional) Indicates whether an Amazon EBS volume is EBS-optimized. Changing this forces a new resource to be created.
* `ebs_config` (Optional) One or more `ebs_config` blocks as defined below. Changing this forces a new resource to be created.

`ebs_config` supports the following:
* `iops` - (Optional) The number of I/O operations per second (IOPS) that the volume supports.
* `size` - (Optional) The volume size, in gibibytes (GiB). This can be a number from 1 - 1024. If the volume type is EBS-optimized, the minimum value is 10.
* `type` - (Optional) The volume type. Valid options are 'gp2', 'io1' and 'standard'.
* `volumes_per_instance` - (Optional) The number of EBS Volumes to attach per instance.

## Attributes Reference

The following attributes are exported:

* `id` - The EMR Instance ID

* `running_instance_count` The number of instances currently running in this instance group.

* `status` The current status of the instance group.
