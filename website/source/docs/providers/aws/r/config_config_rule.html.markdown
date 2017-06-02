---
layout: "aws"
page_title: "AWS: aws_config_config_rule"
sidebar_current: "docs-aws-resource-config-config-rule"
description: |-
  Provides an AWS Config Rule.
---

# aws\_config\_config\_rule

Provides an AWS Config Rule.

~> **Note:** Config Rule requires an existing [Configuration Recorder](/docs/providers/aws/r/config_configuration_recorder.html) to be present. Use of `depends_on` is recommended (as shown below) to avoid race conditions.

## Example Usage

```hcl
resource "aws_config_config_rule" "r" {
  name = "example"

  source {
    owner             = "AWS"
    source_identifier = "S3_BUCKET_VERSIONING_ENABLED"
  }

  depends_on = ["aws_config_configuration_recorder.foo"]
}

resource "aws_config_configuration_recorder" "foo" {
  name     = "example"
  role_arn = "${aws_iam_role.r.arn}"
}

resource "aws_iam_role" "r" {
  name = "my-awsconfig-role"

  assume_role_policy = <<POLICY
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Action": "sts:AssumeRole",
      "Principal": {
        "Service": "config.amazonaws.com"
      },
      "Effect": "Allow",
      "Sid": ""
    }
  ]
}
POLICY
}

resource "aws_iam_role_policy" "p" {
  name = "my-awsconfig-policy"
  role = "${aws_iam_role.r.id}"

  policy = <<POLICY
{
  "Version": "2012-10-17",
  "Statement": [
  	{
  		"Action": "config:Put*",
  		"Effect": "Allow",
  		"Resource": "*"

  	}
  ]
}
POLICY
}
```

## Argument Reference

The following arguments are supported:

* `name` - (Required) The name of the rule
* `description` - (Optional) Description of the rule
* `input_parameters` - (Optional) A string in JSON format that is passed to the AWS Config rule Lambda function.
* `maximum_execution_frequency` - (Optional) The maximum frequency with which AWS Config runs evaluations for a rule.
* `scope` - (Optional) Scope defines which resources can trigger an evaluation for the rule as documented below.
* `source` - (Required) Source specifies the rule owner, the rule identifier, and the notifications that cause
	the function to evaluate your AWS resources as documented below.

### `scope`

Defines which resources can trigger an evaluation for the rule.
If you do not specify a scope, evaluations are triggered when any resource in the recording group changes.

* `compliance_resource_id` - (Optional) The IDs of the only AWS resource that you want to trigger an evaluation for the rule.
	If you specify a resource ID, you must specify one resource type for `compliance_resource_types`.
* `compliance_resource_types` - (Optional) A list of resource types of only those AWS resources that you want to trigger an
	evaluation for the rule. e.g. `AWS::EC2::Instance`. You can only specify one type if you also specify
	a resource ID for `compliance_resource_id`. See [relevant part of AWS Docs](http://docs.aws.amazon.com/config/latest/APIReference/API_ResourceIdentifier.html#config-Type-ResourceIdentifier-resourceType) for available types.
* `tag_key` - (Optional, Required if `tag_value` is specified) The tag key that is applied to only those AWS resources that you want you
	want to trigger an evaluation for the rule.
* `tag_value` - (Optional) The tag value applied to only those AWS resources that you want to trigger an evaluation for the rule.

### `source`

Provides the rule owner (AWS or customer), the rule identifier, and the notifications that cause the function to evaluate your AWS resources.

* `owner` - (Required) Indicates whether AWS or the customer owns and manages the AWS Config rule.
	The only valid value is `AWS` or `CUSTOM_LAMBDA`. Keep in mind that Lambda function will require `aws_lambda_permission` to allow AWSConfig to execute the function.
* `source_identifier` - (Required) For AWS Config managed rules, a predefined identifier from a list. For example,
	`IAM_PASSWORD_POLICY` is a managed rule. To reference a managed rule, see [Using AWS Managed Config Rules](http://docs.aws.amazon.com/config/latest/developerguide/evaluate-config_use-managed-rules.html).
	For custom rules, the identifier is the ARN of the rule's AWS Lambda function, such as `arn:aws:lambda:us-east-1:123456789012:function:custom_rule_name`.
* `source_detail` - (Optional) Provides the source and type of the event that causes AWS Config to evaluate your AWS resources. Only valid if `owner` is `CUSTOM_LAMBDA`.
	* `event_source` - (Optional) The source of the event, such as an AWS service, that triggers AWS Config
		to evaluate your AWS resources. This defaults to `aws.config` and is the only valid value.
	* `maximum_execution_frequency` - (Optional) The frequency that you want AWS Config to run evaluations for a rule that
		is triggered periodically. If specified, requires `message_type` to be `ScheduledNotification`.
	* `message_type` - (Optional) The type of notification that triggers AWS Config to run an evaluation for a rule. You can specify the following notification types:
	    * `ConfigurationItemChangeNotification` - Triggers an evaluation when AWS
	    	Config delivers a configuration item as a result of a resource change.
	    * `OversizedConfigurationItemChangeNotification` - Triggers an evaluation
	    	when AWS Config delivers an oversized configuration item. AWS Config may
	    	generate this notification type when a resource changes and the notification
	    	exceeds the maximum size allowed by Amazon SNS.
	    * `ScheduledNotification` - Triggers a periodic evaluation at the frequency
	    	specified for `maximum_execution_frequency`.
	    * `ConfigurationSnapshotDeliveryCompleted` - Triggers a periodic evaluation
	    	when AWS Config delivers a configuration snapshot.

## Attributes Reference

The following attributes are exported:

* `arn` - The ARN of the config rule
* `rule_id` - The ID of the config rule

## Import

Config Rule can be imported using the name, e.g.

```
$ terraform import aws_config_config_rule.foo example
```
