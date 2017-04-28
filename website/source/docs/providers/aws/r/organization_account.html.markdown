---
layout: "aws"
page_title: "AWS: aws_organization_account
sidebar_current: "docs-aws-resource-organization-account|"
description: |-
  Provides a resource to create a member account in the current organization.
---

# aws\_organization\_account
 
-> **Note:** Account creation must be done from the organization's master account.

-> **Note:** AWS member accounts must be deleted manually by following these steps: 1) Perform a root account password recovery for the email address that was specified for the account in Organizations. 2) Login to the account as that root user. 3) Navigate to "My Organization" in the account menu top-right. 4) Leave the organization. 5) Once the account has successfully left the organization, delete the account as usual.


Provides a resource to create a member account in the current organization.

## Example Usage:

```hcl
resource "aws_organization_account" "account" {
  name  = "my_new_account"
  email = "john@doe.org"
}
```

## Argument Reference

The following arguments are supported:

* `name` - (Required) A friendly name for the member account.
* `email` - (Required) The email address of the owner to assign to the new member account. This email address must not already be associated with another AWS account.
* `iam_user_access_to_billing` - (Optional) If set to ALLOW, the new account enables IAM users to access account billing information if they have the required permissions. If set to DENY, then only the root user of the new account can access account billing information.
* `role_name` - (Optional) The name of an IAM role that Organizations automatically preconfigures in the new member account. This role trusts the master account, allowing users in the master account to assume the role, as permitted by the master account administrator. The role has administrator permissions in the new member account. 

## Import

The AWS member account can be imported by using the `account_id`, e.g.

```
$ terraform import aws_organization_account.my_org 111111111111
```
