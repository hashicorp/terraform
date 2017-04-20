---
layout: "aws"
page_title: "AWS: aws_ip_ranges"
sidebar_current: "docs-aws-datasource-ip_ranges"
description: |-
  Get information on AWS IP ranges.
---

# aws\_ip_ranges

Use this data source to get the [IP ranges][1] of various AWS products and services.

## Example Usage

```hcl
data "aws_ip_ranges" "european_ec2" {
  regions  = ["eu-west-1", "eu-central-1"]
  services = ["ec2"]
}

resource "aws_security_group" "from_europe" {
  name = "from_europe"

  ingress {
    from_port   = "443"
    to_port     = "443"
    protocol    = "tcp"
    cidr_blocks = ["${data.aws_ip_ranges.european_ec2.cidr_blocks}"]
  }

  tags {
    CreateDate = "${data.aws_ip_ranges.european_ec2.create_date}"
    SyncToken  = "${data.aws_ip_ranges.european_ec2.sync_token}"
  }
}
```

## Argument Reference

* `regions` - (Optional) Filter IP ranges by regions (or include all regions, if
omitted). Valid items are `global` (for `cloudfront`) as well as all AWS regions
(e.g. `eu-central-1`)

* `services` - (Required) Filter IP ranges by services. Valid items are `amazon`
(for amazon.com), `cloudfront`, `ec2`, `route53`, `route53_healthchecks` and `S3`.

~> **NOTE:** If the specified combination of regions and services does not yield any
CIDR blocks, Terraform will fail.

## Attributes Reference

* `cidr_blocks` - The lexically ordered list of CIDR blocks.
* `create_date` - The publication time of the IP ranges (e.g. `2016-08-03-23-46-05`).
* `sync_token` - The publication time of the IP ranges, in Unix epoch time format
  (e.g. `1470267965`).

[1]: http://docs.aws.amazon.com/general/latest/gr/aws-ip-ranges.html
