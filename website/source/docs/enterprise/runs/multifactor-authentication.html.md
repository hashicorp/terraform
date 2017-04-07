---
layout: "enterprise"
page_title: "AWS Multi-Factor Authentication - Runs - Terraform Enterprise"
sidebar_current: "docs-enterprise-runs-multifactor-authentication"
description: |-
  Installing custom software on the Terraform Runners.
---

# AWS Multi-Factor Authentication for Terraform Runs in Terraform Enterprise

You can optionally configure Terraform plans and applies to use multi-factor authentication using [AWS Secure Token Service](http://docs.aws.amazon.com/STS/latest/APIReference/Welcome.html).

This option is disabled by default and can be enabled by an organization owner.

!> This is an advanced feature that enables changes to active infrastructure
without user confirmation. Please understand the implications to your
infrastructure before enabling.

## Setting Up AWS Multi-Factor Authentication

Before you are able to set up multi-factor authentication in Terraform
Enterprise, you must set up an IAM user in AWS. More details about creating an
IAM user can be found
[here](http://docs.aws.amazon.com/IAM/latest/UserGuide/id_credentials_mfa_enable.html).
Setting up an AWS IAM user will provide you with the serial number and access
keys that you will need in order to connect to AWS Secure Token Service.

In order to set up multi-factor authentication for your organization, you must
have the following environment variables in your configuration:
'AWS_ACCESS_KEY_ID", "AWS_SECRET_ACCESS_KEY", "AWS_MFA_SERIAL_NUMBER". You can
set these variables at `/settings/organization_variables.`


## Enabling AWS Multi-Factor Authentication

To enable multi-factor authentication, visit the environment settings page:

```text
/terraform/:organization/environments/:environment/settings
```

Use the drop down labeled "AWS Multi-Factor Authentication ". There are
currently three levels available: "never", "applies only", and "plans and
applies". Once you have selected your desired level, save your settings. All
subsequent runs on the environment will now require the selected level of
authentication.

## Using AWS Multi-Factor Authentication

Once you have elected to use AWS MFA for your Terraform Runs, you will then be
prompted to enter a token code each time you plan or apply the run depending on
your settings. Your one time use token code will be sent to you via the method
you selected when setting up your
[IAM account](http://docs.aws.amazon.com/IAM/latest/UserGuide/id_credentials_mfa_enable.html).

If you have selected "applies only", you will be able to queue and run a plan
without entering your token code. Once the run finishes, you will need to enter
your token code and click "Authenticate" before the applying the plan. Once you
submit your token code, the apply will start, and you will see "Authenticated
with MFA by `user`" in the UI. If for any case there is an error when submitting
your token code, the lock icon in the UI will turn red, and an error will appear
alerting you to the failure.

If you have selected "plans and applies", you will be prompted to enter your
token before queueing your plan.  Once you enter the token and click
"Authenticate", you will see "Authenticated with MFA by `user`" appear in the UI
logs. The plan will queue and you may run the plan once it is queued. Then,
before applying, you will be asked to authenticate with MFA again. Enter your
token, click Authenticate, and note that "Authenticated with MFA by `user`"
appears in the UI log after the apply begins. If for any case there is an error
authenticating, the lock icon in the UI will turn red, and an error will appear
alerting you to the failure.

## Using AWS Multi-Factor Authentication with AWS STS AssumeRole

The AWS Secure Token Service can be used to return a set of temporary security
credentials that a user can use to access resources that they might not normally
have access to (known as AssumeRole). The AssumeRole workflow is compatible with
AWS multi-factor authentication in Terraform Enterprise.

To use AssumeRole, you first need to create an IAM role and edit the trust
relationship policy document to contain the following:

```json
    {
  "Version": "2012-10-17",
  "Statement": [
    {
      "Effect": "Allow",
      "Principal": {
        "AWS": "arn:aws:iam::[INT]:user/[USER]"
      },
      "Action": "sts:AssumeRole",
      "Condition": {
        "Bool": {
          "aws:MultiFactorAuthPresent": "true"
        }
      }
    }
  ]
}
```

You can then configure the Terraform AWS provider to assume a given role by specifying the role ARN within the nested assume_role block:

```hcl
provider "aws" {
  # ...

  assume_role {
    role_arn = "arn:aws:iam::[INT]:role/[ROLE]"
  }
}
```
