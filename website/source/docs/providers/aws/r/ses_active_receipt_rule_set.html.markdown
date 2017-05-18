---
layout: "aws"
page_title: "AWS: ses_active_receipt_rule_set"
sidebar_current: "docs-aws-resource-ses-active-receipt-rule-set"
description: |-
  Provides a resource to designate the active SES receipt rule set
---

# aws\_ses\_active_receipt_rule_set

Provides a resource to designate the active SES receipt rule set

## Example Usage

```hcl
resource "aws_ses_active_receipt_rule_set" "main" {
  rule_set_name = "primary-rules"
}
```

## Argument Reference

The following arguments are supported:

* `rule_set_name` - (Required) The name of the rule set
