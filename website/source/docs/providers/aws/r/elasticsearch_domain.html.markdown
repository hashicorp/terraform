---
layout: "aws"
page_title: "AWS: aws_elasticsearch_domain"
sidebar_current: "docs-aws-elasticsearch-domain"
description: |-
  Provides an ElasticSearch Domain.
---

# aws\_elasticsearch\_domain


## Example Usage

```
resource "aws_elasticsearch_domain" "es" {
	domain_name = "tf-test"
	advanced_options {
		"rest.action.multi.allow_explicit_index" = true
	}

	access_policies = <<CONFIG
{
	"Version": "2012-10-17",
	"Statement": [
		{
			"Action": "es:*",
			"Principal": "*",
			"Effect": "Allow",
			"Condition": {
				"IpAddress": {"aws:SourceIp": ["66.193.100.22/32"]}
			}
		}
	]
}
CONFIG

	snapshot_options {
		automated_snapshot_start_hour = 23
	}
}
```

## Argument Reference

The following arguments are supported:

* `domain_name` - (Required) Name of the domain.
* `access_policies` - (Optional) IAM policy document specifying the access policies for the domain
* `advanced_options` - (Optional) Key-value string pairs to specify advanced configuration options.
* `ebs_options` - (Optional) EBS related options, see below.
* `cluster_config` - (Optional) Cluster configuration of the domain, see below.
* `snapshot_options` - (Optional) Snapshot related options, see below.

**ebs_options** supports the following attributes:

* `ebs_enabled` - (Required) Whether EBS volumes are attached to data nodes in the domain
* `volume_type` - (Optional) The type of EBS volumes attached to data nodes.
* `volume_size` - (Optional) The size of EBS volumes attached to data nodes.
* `iops` - (Optional) The baseline input/output (I/O) performance of EBS volumes
	attached to data nodes. Applicable only for the Provisioned IOPS EBS volume type.

**cluster_config** supports the following attributes:

* `instance_type` - (Optional) Instance type of data nodes in the cluster.
* `instance_count` - (Optional) Number of instances in the cluster.
* `dedicated_master_enabled` - (Optional) Indicates whether dedicated master nodes are enabled for the cluster.
* `dedicated_master_type` - (Optional) Instance type of the dedicated master nodes in the cluster.
* `dedicated_master_count` - (Optional) Number of dedicated master nodes in the cluster
* `zone_awareness_enabled` - (Optional) Indicates whether zone awareness is enabled.

**snapshot_options** supports the following attribute:

* `automated_snapshot_start_hour` - (Required) Hour during which the service takes an automated daily
	snapshot of the indices in the domain.


## Attributes Reference

The following attributes are exported:

* `arn` - Amazon Resource Name (ARN) of the domain.
* `domain_id` - Unique identifier for the domain.
* `endpoint` - Domain-specific endpoint used to submit index, search, and data upload requests.
