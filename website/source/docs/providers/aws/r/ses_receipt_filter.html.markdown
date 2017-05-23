---
layout: "aws"
page_title: "AWS: ses_receipt_filter"
sidebar_current: "docs-aws-resource-ses-receipt-filter"
description: |-
  Provides an SES receipt filter
---

# aws\_ses\_receipt_filter

Provides an SES receipt filter resource

## Example Usage

```hcl
resource "aws_ses_receipt_filter" "filter" {
  name   = "block-spammer"
  cidr   = "10.10.10.10"
  policy = "Block"
}
```

## Argument Reference

The following arguments are supported:

* `name` - (Required) The name of the filter
* `cidr` - (Required) The IP address or address range to filter, in CIDR notation
* `policy` - (Required) Block or Allow
