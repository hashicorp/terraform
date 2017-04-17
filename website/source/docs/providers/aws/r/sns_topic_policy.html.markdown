---
layout: "aws"
page_title: "AWS: sns_topic_policy"
sidebar_current: "docs-aws-resource-sns-topic-policy"
description: |-
  Provides an SNS topic policy resource.
---

# aws\_sns\_topic\_policy

Provides an SNS topic policy resource

## Example Usage

```hcl
resource "aws_sns_topic" "test" {
  name = "my-topic-with-policy"
}

resource "aws_sns_topic_policy" "custom" {
  arn = "${aws_sns_topic.test.arn}"

  policy = <<POLICY
{
  "Version": "2012-10-17",
  "Id": "default",
  "Statement":[{
    "Sid": "default",
    "Effect": "Allow",
    "Principal": {"AWS":"*"},
    "Action": [
      "SNS:GetTopicAttributes",
      "SNS:SetTopicAttributes",
      "SNS:AddPermission",
      "SNS:RemovePermission",
      "SNS:DeleteTopic"
    ],
    "Resource": "${aws_sns_topic.test.arn}"
  }]
}
POLICY
}
```

## Argument Reference

The following arguments are supported:

* `arn` - (Required) The ARN of the SNS topic
* `policy` - (Required) The fully-formed AWS policy as JSON
