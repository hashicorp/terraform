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

```hcl
# Configure the AWS Provider
provider "aws" {
  access_key = "${var.aws_access_key}"
  secret_key = "${var.aws_secret_key}"
  region     = "us-east-1"
}

# Create a web server
resource "aws_instance" "web" {
  # ...
}
```

## Authentication

The AWS provider offers a flexible means of providing credentials for
authentication. The following methods are supported, in this order, and
explained below:

- Static credentials
- Environment variables
- Shared credentials file
- EC2 Role

### Static credentials ###

Static credentials can be provided by adding an `access_key` and `secret_key` in-line in the
AWS provider block:

Usage:

```hcl
provider "aws" {
  region     = "us-west-2"
  access_key = "anaccesskey"
  secret_key = "asecretkey"
}
```

### Environment variables

You can provide your credentials via the `AWS_ACCESS_KEY_ID` and
`AWS_SECRET_ACCESS_KEY`, environment variables, representing your AWS
Access Key and AWS Secret Key, respectively.  The `AWS_DEFAULT_REGION`
and `AWS_SESSION_TOKEN` environment variables are also used, if
applicable:

```hcl
provider "aws" {}
```

Usage:

```hcl
$ export AWS_ACCESS_KEY_ID="anaccesskey"
$ export AWS_SECRET_ACCESS_KEY="asecretkey"
$ export AWS_DEFAULT_REGION="us-west-2"
$ terraform plan
```

### Shared Credentials file

You can use an AWS credentials file to specify your credentials. The
default location is `$HOME/.aws/credentials` on Linux and OS X, or
`"%USERPROFILE%\.aws\credentials"` for Windows users. If we fail to
detect credentials inline, or in the environment, Terraform will check
this location. You can optionally specify a different location in the
configuration by providing the `shared_credentials_file` attribute, or
in the environment with the `AWS_SHARED_CREDENTIALS_FILE` variable. This
method also supports a `profile` configuration and matching
`AWS_PROFILE` environment variable:

Usage:

```hcl
provider "aws" {
  region                  = "us-west-2"
  shared_credentials_file = "/Users/tf_user/.aws/creds"
  profile                 = "customprofile"
}
```

### EC2 Role

If you're running Terraform from an EC2 instance with IAM Instance Profile
using IAM Role, Terraform will just ask
[the metadata API](http://docs.aws.amazon.com/AWSEC2/latest/UserGuide/iam-roles-for-amazon-ec2.html#instance-metadata-security-credentials)
endpoint for credentials.

This is a preferred approach over any other when running in EC2 as you can avoid
hard coding credentials. Instead these are leased on-the-fly by Terraform
which reduces the chance of leakage.

You can provide the custom metadata API endpoint via the `AWS_METADATA_ENDPOINT` variable
which expects the endpoint URL, including the version, and defaults to `http://169.254.169.254:80/latest`.

### Assume role

If provided with a role ARN, Terraform will attempt to assume this role
using the supplied credentials.

Usage:

```hcl
provider "aws" {
  assume_role {
    role_arn     = "arn:aws:iam::ACCOUNT_ID:role/ROLE_NAME"
    session_name = "SESSION_NAME"
    external_id  = "EXTERNAL_ID"
  }
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

* `assume_role` - (Optional) An `assume_role` block (documented below). Only one
  `assume_role` block may be in the configuration.

* `shared_credentials_file` = (Optional) This is the path to the shared credentials file.
  If this is not set and a profile is specified, `~/.aws/credentials` will be used.

* `token` - (Optional) Use this to set an MFA token. It can also be sourced
  from the `AWS_SESSION_TOKEN` environment variable.

* `max_retries` - (Optional) This is the maximum number of times an API
  call is retried, in the case where requests are being throttled or
  experiencing transient failures. The delay between the subsequent API
  calls increases exponentially.

* `allowed_account_ids` - (Optional) List of allowed, white listed, AWS
  account IDs to prevent you from mistakenly using an incorrect one (and
  potentially end up destroying a live environment). Conflicts with
  `forbidden_account_ids`.

* `forbidden_account_ids` - (Optional) List of forbidden, blacklisted,
  AWS account IDs to prevent you mistakenly using a wrong one (and
  potentially end up destroying a live environment). Conflicts with
  `allowed_account_ids`.

* `insecure` - (Optional) Explicitly allow the provider to
  perform "insecure" SSL requests. If omitted, default value is `false`.

* `skip_credentials_validation` - (Optional) Skip the credentials
  validation via the STS API. Useful for AWS API implementations that do
  not have STS available or implemented.

* `skip_get_ec2_platforms` - (Optional) Skip getting the supported EC2
  platforms. Used by users that don't have ec2:DescribeAccountAttributes
  permissions.

* `skip_region_validation` - (Optional) Skip validation of provided region name.
  Useful for AWS-like implementations that use their own region names
  or to bypass the validation for regions that aren't publicly available yet.

* `skip_requesting_account_id` - (Optional) Skip requesting the account
  ID.  Useful for AWS API implementations that do not have the IAM, STS
  API, or metadata API.  When set to `true`, prevents you from managing
  any resource that requires Account ID to construct an ARN, e.g.
  - `aws_db_instance`
  - `aws_db_option_group`
  - `aws_db_parameter_group`
  - `aws_db_security_group`
  - `aws_db_subnet_group`
  - `aws_elasticache_cluster`
  - `aws_glacier_vault`
  - `aws_rds_cluster`
  - `aws_rds_cluster_instance`
  - `aws_rds_cluster_parameter_group`
  - `aws_redshift_cluster`

* `skip_metadata_api_check` - (Optional) Skip the AWS Metadata API
  check.  Useful for AWS API implementations that do not have a metadata
  API endpoint.  Setting to `true` prevents Terraform from authenticating
  via the Metadata API. You may need to use other authentication methods
  like static credentials, configuration variables, or environment
  variables.

* `s3_force_path_style` - (Optional) Set this to `true` to force the
  request to use path-style addressing, i.e.,
  `http://s3.amazonaws.com/BUCKET/KEY`. By default, the S3 client will use
  virtual hosted bucket addressing, `http://BUCKET.s3.amazonaws.com/KEY`,
  when possible. Specific to the Amazon S3 service.

The nested `assume_role` block supports the following:

* `role_arn` - (Required) The ARN of the role to assume.

* `session_name` - (Optional) The session name to use when making the
  AssumeRole call.

* `external_id` - (Optional) The external ID to use when making the
  AssumeRole call.

* `policy` - (Optional) A more restrictive policy to apply to the temporary credentials.
This gives you a way to further restrict the permissions for the resulting temporary
security credentials. You cannot use the passed policy to grant permissions that are
in excess of those allowed by the access policy of the role that is being assumed.

Nested `endpoints` block supports the following:

* `cloudwatch` - (Optional) Use this to override the default endpoint
  URL constructed from the `region`. It's typically used to connect to
  custom CloudWatch endpoints.

* `cloudwatchevents` - (Optional) Use this to override the default endpoint
  URL constructed from the `region`. It's typically used to connect to
  custom CloudWatchEvents endpoints.

* `cloudwatchlogs` - (Optional) Use this to override the default endpoint
  URL constructed from the `region`. It's typically used to connect to
  custom CloudWatchLogs endpoints.

* `cloudformation` - (Optional) Use this to override the default endpoint
  URL constructed from the `region`. It's typically used to connect to
  custom CloudFormation endpoints.

* `dynamodb` - (Optional) Use this to override the default endpoint
  URL constructed from the `region`. It's typically used to connect to
  `dynamodb-local`.

* `kinesis` - (Optional) Use this to override the default endpoint
  URL constructed from the `region`. It's typically used to connect to
  `kinesalite`.

* `kms` - (Optional) Use this to override the default endpoint
  URL constructed from the `region`. It's typically used to connect to
  custom KMS endpoints.

* `iam` - (Optional) Use this to override the default endpoint
  URL constructed from the `region`. It's typically used to connect to
  custom IAM endpoints.

* `ec2` - (Optional) Use this to override the default endpoint
  URL constructed from the `region`. It's typically used to connect to
  custom EC2 endpoints.

* `elb` - (Optional) Use this to override the default endpoint
  URL constructed from the `region`. It's typically used to connect to
  custom ELB endpoints.

* `rds` - (Optional) Use this to override the default endpoint
  URL constructed from the `region`. It's typically used to connect to
  custom RDS endpoints.

* `s3` - (Optional) Use this to override the default endpoint
  URL constructed from the `region`. It's typically used to connect to
  custom S3 endpoints.

* `sns` - (Optional) Use this to override the default endpoint
  URL constructed from the `region`. It's typically used to connect to
  custom SNS endpoints.

* `sqs` - (Optional) Use this to override the default endpoint
  URL constructed from the `region`. It's typically used to connect to
  custom SQS endpoints.

## Getting the Account ID

If you use either `allowed_account_ids` or `forbidden_account_ids`,
Terraform uses several approaches to get the actual account ID
in order to compare it with allowed or forbidden IDs.

Approaches differ per authentication providers:

 * EC2 instance w/ IAM Instance Profile - [Metadata API](http://docs.aws.amazon.com/AWSEC2/latest/UserGuide/ec2-instance-metadata.html)
    is always used. Introduced in Terraform `0.6.16`.
 * All other providers (environment variable, shared credentials file, ...)
    will try two approaches in the following order
   * `iam:GetUser` - Typically useful for IAM Users. It also means
      that each user needs to be privileged to call `iam:GetUser` for themselves.
   * `sts:GetCallerIdentity` - _Should_ work for both IAM Users and federated IAM Roles,
      introduced in Terraform `0.6.16`.
   * `iam:ListRoles` - This is specifically useful for IdP-federated profiles
      which cannot use `iam:GetUser`. It also means that each federated user
      need to be _assuming_ an IAM role which allows `iam:ListRoles`.
      Used in Terraform `0.6.16+`.
      There used to be no better way to get account ID out of the API
      when using federated account until `sts:GetCallerIdentity` was introduced.
