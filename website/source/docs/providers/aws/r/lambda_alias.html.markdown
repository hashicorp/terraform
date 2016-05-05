---
layout: "aws"
page_title: "AWS: aws_lambda_alias"
sidebar_current: "docs-aws-resource-lambda-alias"
description: |-
  Creates a Lambda function alias.
---

# aws\_lambda\_alias

Creates a Lambda function alias. Creates an alias that points to the specified Lambda function version.

For information about Lambda and how to use it, see [What is AWS Lambda?][1]
For information about function aliases, see [CreateAlias][2] in the API docs.

## Example Usage

```
resource "aws_lambda_alias" "test_alias" {
		name = "testalias"
		description = "a sample description"
		function_name = "${aws_lambda_function.lambda_function_test.arn}"
		function_version = "$LATEST"
}
```

## Argument Reference

* `name` - (Required) Name for the alias you are creating. Pattern: `(?!^[0-9]+$)([a-zA-Z0-9-_]+)`
* `description` - (Optional) Description of the alias.
* `function_name` - (Required) The function ARN of the Lambda function for which you want to create an alias.
* `function_version` - (Required) Lambda function version for which you are creating the alias. Pattern: `(\$LATEST|[0-9]+)`.

[1]: http://docs.aws.amazon.com/lambda/latest/dg/welcome.html
[2]: http://docs.aws.amazon.com/lambda/latest/dg/API_CreateAlias.html
