---
layout: "aws"
page_title: "AWS: ses_receipt_rule_set"
sidebar_current: "docs-aws-resource-ses-receipt-rule-set"
description: |-
  Provides an SES receipt rule set resource
---

# aws\_ses\_receipt_rule_set

Provides an SES receipt rule set resource

## Example Usage

```hcl
resource "aws_ses_receipt_rule_set" "main" {
  rule_set_name = "primary-rules"
}
```

## Argument Reference

The following arguments are supported:

* `rule_set_name` - (Required) The name of the rule set
