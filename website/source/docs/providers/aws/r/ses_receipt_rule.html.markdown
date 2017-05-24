---
layout: "aws"
page_title: "AWS: ses_receipt_rule"
sidebar_current: "docs-aws-resource-ses-receipt-rule"
description: |-
  Provides an SES receipt rule resource
---

# aws\_ses\_receipt_rule

Provides an SES receipt rule resource

## Example Usage

```hcl
# Add a header to the email and store it in S3
resource "aws_ses_receipt_rule" "store" {
  name          = "store"
  rule_set_name = "default-rule-set"
  recipients    = ["karen@example.com"]
  enabled       = true
  scan_enabled  = true

  add_header_action {
    header_name  = "Custom-Header"
    header_value = "Added by SES"
  }

  s3_action {
    bucket_name = "emails"
  }
}
```

## Argument Reference

The following arguments are supported:

* `name` - (Required) The name of the rule
* `rule_set_name` - (Required) The name of the rule set
* `after` - (Optional) The name of the rule to place this rule after
* `enabled` - (Optional) If true, the rule will be enabled
* `recipients` - (Optional) A list of email addresses
* `scan_enabled` - (Optional) If true, incoming emails will be scanned for spam and viruses
* `tls_policy` - (Optional) Require or Optional
* `add_header_action` - (Optional) A list of Add Header Action blocks. Documented below.
* `bounce_action` - (Optional) A list of Bounce Action blocks. Documented below.
* `lambda_action` - (Optional) A list of Lambda Action blocks. Documented below.
* `s3_action` - (Optional) A list of S3 Action blocks. Documented below.
* `sns_action` - (Optional) A list of SNS Action blocks. Documented below.
* `stop_action` - (Optional) A list of Stop Action blocks. Documented below.
* `workmail_action` - (Optional) A list of WorkMail Action blocks. Documented below.

Add header actions support the following:

* `header_name` - (Required) The name of the header to add
* `header_value` - (Required) The value of the header to add
* `position` - (Required) The position of the action in the receipt rule

Bounce actions support the following:

* `message` - (Required) The message to send
* `sender` - (Required) The email address of the sender
* `smtp_reply_code` - (Required) The RFC 5321 SMTP reply code
* `status_code` - (Optional) The RFC 3463 SMTP enhanced status code
* `topic_arn` - (Optional) The ARN of an SNS topic to notify
* `position` - (Required) The position of the action in the receipt rule

Lambda actions support the following:

* `function_arn` - (Required) The ARN of the Lambda function to invoke
* `invocation_type` - (Optional) Event or RequestResponse
* `topic_arn` - (Optional) The ARN of an SNS topic to notify
* `position` - (Required) The position of the action in the receipt rule

S3 actions support the following:

* `bucket_name` - (Required) The name of the S3 bucket
* `kms_key_arn` - (Optional) The ARN of the KMS key
* `object_key_prefix` - (Optional) The key prefix of the S3 bucket
* `topic_arn` - (Optional) The ARN of an SNS topic to notify
* `position` - (Required) The position of the action in the receipt rule

SNS actions support the following:

* `topic_arn` - (Required) The ARN of an SNS topic to notify
* `position` - (Required) The position of the action in the receipt rule

Stop actions support the following:

* `scope` - (Required) The scope to apply
* `topic_arn` - (Optional) The ARN of an SNS topic to notify
* `position` - (Required) The position of the action in the receipt rule

WorkMail actions support the following:

* `organization_arn` - (Required) The ARN of the WorkMail organization
* `topic_arn` - (Optional) The ARN of an SNS topic to notify
* `position` - (Required) The position of the action in the receipt rule
