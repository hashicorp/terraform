---
layout: "aws"
page_title: "AWS: aws_emr_instance_group"
sidebar_current: "docs-aws-resource-emr-instance-group"
description: |-
  Provides an Elastic MapReduce Cluster Instance Group
---

# aws\_emr\_instance\_group

Provides an Elastic MapReduce Cluster Instance Group configuration. 
See [Amazon Elastic MapReduce Documentation](http://docs.aws.amazon.com/en_en/ElasticMapReduce/latest/ManagementGuide/InstanceGroups.html) 
for more information. 

~> **NOTE:** At this time, Instance Groups cannot be destroyed through the API nor
web interface. Instance Groups are destroyed when the EMR Cluster is destroyed.
Terraform will resize any Instance Group to zero when destroying the resource.

## Example Usage

```
resource "aws_emr_cluster_instance_group" "task" {
  cluster_id     = "${aws_emr_cluster.tf-test-cluster.id}"
  instance_count = 1
  instance_type  = "m3.xlarge"
  name           = "my little instance group"
}
```

## Argument Reference

The following arguments are supported:

* `name` - (Optional) Optional human friendly name for this Instance Group
* `cluster_id` - (Required) ID of the EMR Cluster to attach to
* `instance_type` - (Required) Type of instances for this Group
* `instance_count` - (Optional) Count of instances to launch



## ec2\_attributes

Attributes for the Instance Group

* `name` - Human friendly name for this Instance Group
* `cluster_id` - ID of the EMR Cluster the group is attached to
* `instance_type` - Type of instances for this Group
* `instance_count` - Count of desired instances to launch
* `running_instance_count` - Count of actual running instances in the group
* `status` - State of the instance group. One of `PROVISIONING`, `BOOTSTRAPPING`, `RUNNING`, `RESIZING`, `SUSPENDED`, `TERMINATING`, `TERMINATED`, `ARRESTED`, `SHUTTING_DOWN`, `ENDED`
