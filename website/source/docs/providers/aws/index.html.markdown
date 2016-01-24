---
layout: "aws"
page_title: "Provider: AWS"
sidebar_current: "docs-aws-index"
description: |-
  The Amazon Web Services (AWS) provider is used to interact with the many resources supported by AWS. The provider needs to be configured with the proper credentials before it can be used.
---

# AWS Provider

The Amazon Web Services (AWS) provider is used to interact with the
many resources supported by AWS. The provider needs to be configured
with the proper credentials before it can be used.

Use the navigation to the left to read about the available resources.

## Example Usage

```
# Configure the AWS Provider
provider "aws" {
    access_key = "${var.aws_access_key}"
    secret_key = "${var.aws_secret_key}"
    region = "us-east-1"
}

# Create a web server
resource "aws_instance" "web" {
    ...
}
```

## Argument Reference

The following arguments are supported in the `provider` block:

* `access_key` - (Optional) This is the AWS access key. It must be provided, but
  it can also be sourced from the `AWS_ACCESS_KEY_ID` environment variable, or via
  a shared credentials file if `profile` is specified.

* `secret_key` - (Optional) This is the AWS secret key. It must be provided, but
  it can also be sourced from the `AWS_SECRET_ACCESS_KEY` environment variable, or
  via a shared credentials file if `profile` is specified.

* `region` - (Required) This is the AWS region. It must be provided, but
  it can also be sourced from the `AWS_DEFAULT_REGION` environment variables, or
  via a shared credentials file if `profile` is specified.

* `profile` - (Optional) This is the AWS profile name as set in the shared credentials
  file.

* `shared_credentials_file` = (Optional) This is the path to the shared credentials file.
  If this is not set and a profile is specified, ~/.aws/credentials will be used.

* `token` - (Optional) Use this to set an MFA token. It can also be sourced
  from the `AWS_SECURITY_TOKEN` environment variable.

* `max_retries` - (Optional) This is the maximum number of times an API call is
  being retried in case requests are being throttled or experience transient failures.
  The delay between the subsequent API calls increases exponentially.

* `allowed_account_ids` - (Optional) List of allowed AWS account IDs (whitelist)
  to prevent you mistakenly using a wrong one (and end up destroying live environment).
  Conflicts with `forbidden_account_ids`.

* `forbidden_account_ids` - (Optional) List of forbidden AWS account IDs (blacklist)
  to prevent you mistakenly using a wrong one (and end up destroying live environment).
  Conflicts with `allowed_account_ids`.

* `dynamodb_endpoint` - (Optional) Use this to override the default endpoint
  URL constructed from the `region`. It's typically used to connect to
  dynamodb-local.

* `kinesis_endpoint` - (Optional) Use this to override the default endpoint URL
  constructed from the `region`. It's typically used to connect to kinesalite.

