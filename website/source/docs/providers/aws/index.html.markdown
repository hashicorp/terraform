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

## Authentication 

The AWS provider offers flexible means of providing credentials for
authentication. The following methods are supported, in this order, and
explained below:

- Static credentials
- Environment variables
- Shared credentials file


### Static credentials ###

Static credentials can be provided by adding an `access_key` and `secret_key` in-line in the
aws provider block:

Usage: 

```
provider "aws" {
  region     = "us-west-2"
  access_key = "anaccesskey"
  secret_key = "asecretkey"
}
```

###Environment variables

You can provide your credentials via `AWS_ACCESS_KEY_ID` and `AWS_SECRET_ACCESS_KEY`, 
environment variables, representing your AWS Access Key and AWS Secret Key, respectively.
`AWS_DEFAULT_REGION` and `AWS_SECURITY_TOKEN` are also used, if applicable:

```
provider "aws" {}
```

Usage:

```
$ export AWS_ACCESS_KEY_ID="anaccesskey" 
$ export AWS_SECRET_ACCESS_KEY="asecretkey"
$ export AWS_DEFAULT_REGION="us-west-2"
$ terraform plan
```

###Shared Credentials file

You can use an AWS credentials file to specify your credentials. The default
location is `$HOME/.aws/credentials` on Linux and OSX, or `"%USERPROFILE%\.aws\credentials"` 
for Windows users. If we fail to detect credentials inline, or in the
environment, Terraform will check this location. You can optionally specify a
different location in the configuration by providing `shared_credentials_file`,
or in the environment with the `AWS_SHARED_CREDENTIALS_FILE` variable. This
method also supports a `profile` configuration and matching `AWS_PROFILE`
environment variable:

Usage: 

```
provider "aws" {
  region                   = "us-west-2"
  shared_credentials_file  = "/Users/tf_user/.aws/creds"
  profile                  = "customprofile"
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

* `insecure` - (Optional) Optional) Explicitly allow the provider to
  perform "insecure" SSL requests. If omitted, default value is `false`

* `dynamodb_endpoint` - (Optional) Use this to override the default endpoint
  URL constructed from the `region`. It's typically used to connect to
  dynamodb-local.

* `kinesis_endpoint` - (Optional) Use this to override the default endpoint
  URL constructed from the `region`. It's typically used to connect to
  kinesalite.

Nested `endpoints` block supports the followings:

* `iam` - (Optional) Use this to override the default endpoint
  URL constructed from the `region`. It's typically used to connect to
  custom iam endpoints.

* `ec2` - (Optional) Use this to override the default endpoint
  URL constructed from the `region`. It's typically used to connect to
  custom ec2 endpoints.

* `elb` - (Optional) Use this to override the default endpoint
  URL constructed from the `region`. It's typically used to connect to
  custom elb endpoints.