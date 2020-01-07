---
layout: "backend-types"
page_title: "Backend Type: dynamodb"
sidebar_current: "docs-backends-types-standard-dynamodb"
description: |-
  Terraform can both store state remotely and lock that state with DynamoDB.
---

# DynamoDB

**Kind: Standard (with locking via DynamoDB)**

Stores the state in a given table on [Amazon DynamoDB](https://aws.amazon.com/dynamodb/).
This backend supports 
* state locking and consistency checking via Amazon DynamoDB, which can be enabled by setting the `lock_table` field to an existing DynamoDB table name.
* global-state can be enabled by setting the `lock_table` and `state_table` fields to an existing DynamoDB global table names.

~> **Warning!** It is highly recommended that you enable versioning using `state_days_ttl` to allow 
for state recovery in the case of accidental deletions and human error.

## Example Configuration without lock

```hcl
terraform {
  backend "dynamodb" {
    state_table = "mytable"
    hash        = "hash_key"
    region      = "eu-west-1"
  }
}
```

This assumes we have a table created called `mytable`. The
Terraform state is written into table using `hash_key` as partition key.

Note that for the access credentials we recommend using a
[partial configuration](/docs/backends/config.html).

## Example Configuration with lock

```hcl
terraform {
  backend "dynamodb" {
    state_table = "mytable"
    hash        = "hash_key"
    lock_table  = "mylock"
    region      = "eu-west-1"
  }
}
```

This assumes we have a table created called `mytable` and a table called `mylock`. 
The Terraform state is written into table using `hash_key` as partition key. You 
can use same lock table with many state tables, consistency check information are
saved in lock tables using `mytable/hash_key` as partition key.

Note that for the access credentials we recommend using a
[partial configuration](/docs/backends/config.html).

## Example Configuration with global tables

```hcl
terraform {
  backend "dynamodb" {
    state_table = "mytable"
    hash        = "hash_key"
    lock_table  = "mylock"
  }
}
```

Stores the state and locking info on [Amazon Global DynamoDB](https://docs.aws.amazon.com/amazondynamodb/latest/developerguide/GlobalTables.html).
This assumes we have a global table created called `mytable` and a global table called `mylock`.
Before start to update your infrastructure the backend ensures that global lock is performed. 

Use `AWS_REGION` or `AWS_DEFAULT_REGION` envs instead of `region` variable.

### DynamoDB Tables Permissions

Terraform will need the following AWS IAM permissions on the DynamoDB 
table (`arn:aws:dynamodb:::table/mytable`):

* `dynamodb:GetItem`
* `dynamodb:PutItem`
* `dynamodb:DeleteItem`

This is seen in the following AWS IAM Statement:

```json
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Effect": "Allow",
      "Action": [
        "dynamodb:GetItem",
        "dynamodb:PutItem",
        "dynamodb:DeleteItem"
      ],
      "Resource": "arn:aws:dynamodb:*:*:table/mytable"
    }
  ]
}
```

If you are using state locking, Terraform will need the same AWS IAM
permissions but also on the DynamoDB lock table (`arn:aws:dynamodb:::table/mylock`).
This is seen in the following AWS IAM Statement:

```json
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Effect": "Allow",
      "Action": [
        "dynamodb:GetItem",
        "dynamodb:PutItem",
        "dynamodb:DeleteItem"
      ],
      "Resource": [
        "arn:aws:dynamodb:*:*:table/mytable",
        "arn:aws:dynamodb:*:*:table/mylock",
      ]
    }
  ]
}
```
If you are using global tables, Terraform will need also the following AWS IAM permissions 
on the DynamoDB table (`arn:aws:dynamodb:::table/mytable`):

* `dynamodb:DescribeGlobalTable`
* `dynamodb:ListGlobalTables`

This is seen in the following AWS IAM Statement:

```json
{
  "Version": "2012-10-17",
  "Statement": [
    {
        "Effect": "Allow",
        "Action": "dynamodb:DescribeGlobalTable",
        "Resource": [
          "arn:aws:dynamodb:*:*:global-table/mytable",
          "arn:aws:dynamodb:*:*:global-table/mylock",
        ]
    },
    {
        "Effect": "Allow",
        "Action": "dynamodb:ListGlobalTables",
        "Resource": "*"
    },
    {
        "Effect": "Allow",
        "Action": [
          "dynamodb:GetItem",
          "dynamodb:PutItem",
          "dynamodb:DeleteItem",
        ],
        "Resource": [
          "arn:aws:dynamodb:*:*:table/mytable",
          "arn:aws:dynamodb:*:*:table/mylock",
        ]
    }
  ]
}
```

## Using the DynamoDB remote state

To make use of the DynamoDB remote state we can use the
[`terraform_remote_state` data
source](/docs/providers/terraform/d/remote_state.html).

```hcl
data "terraform_remote_state" "network" {
  backend = "DynamoDB"
  config = {
    state_table = "mytable"
    hash        = "network"
  }
}
```

The `terraform_remote_state` data source will return all of the root module
outputs defined in the referenced remote state (but not any outputs from
nested modules unless they are explicitly output again in the root). An
example output might look like:

```
data.terraform_remote_state.network:
  id = 2016-10-29 01:57:59.780010914 +0000 UTC
  addresses.# = 2
  addresses.0 = 52.207.220.222
  addresses.1 = 54.196.78.166
  backend = DynamoDB
  config.% = 3
  config.table = terraform-state-prod
  config.key = network/terraform.tfstate
  config.region = us-east-1
  elb_address = web-elb-790251200.us-east-1.elb.amazonaws.com
  public_subnet_id = subnet-1e05dd33
```

## Configuration variables

The following configuration options or environment variables are supported:

 * `state_table` - (Required) The name of the DynamoDB table. The table must have a hash key named StateID (string) and sort key named SegmentID (number).
 * `hash` - (Required) The hash key used to save state inside the table. When using
   a non-default [workspace](/docs/state/workspaces.html), the state hash will
   be `workspace_key_prefix=workspace_name/hash`
 * `compression` - (Optional) Enable state compression using gzip. You can enable/disable this feature at any time. This defaults to False.
 * `global_table_health_check` - (Optional) Enable global table health check. You can use this backend to deploy a disaster recovery solution. When the feature is enabled a test writing into DynamoDB global table is performed to assess region availability, if the check fails state locking for the unhealthy regions are skipped. This defaults to True.
 * `state_days_ttl` - (Optional) Enable state versioning. By default, the new state overrides the old one, by setting state_days_ttl a new item is created for each update. The latest state is the ones with the greatest VersionID value. You can configure state expiration by setting a value greater than zero in `state_days_ttl` variable and enabling [DynamoDB TTL](https://docs.aws.amazon.com/amazondynamodb/latest/developerguide/TTL.html). You can enable/disable this feature at any time, if versioning was enabled old states are retained until expiration. TTL=0 in the DynamoDB table means that state does not expire, `state_days_ttl=0` creates a new version but does not expires the state.  
 * `region` / `AWS_REGION` / `AWS_DEFAULT_REGION` - (Optional) The region of the DynamoDB table. Use ENVs when deploy a disaster recovery solution.
 * `endpoint` / `AWS_DynamoDB_ENDPOINT` - (Optional) A custom endpoint for the
 DynamoDB API.
 * `access_key` / `AWS_ACCESS_KEY_ID` - (Optional) AWS access key.
 * `secret_key` / `AWS_SECRET_ACCESS_KEY` - (Optional) AWS secret access key.
 * `lock_table` - (Optional) The name of a DynamoDB table to use for state
   locking and consistency. The table must have a primary key named LockID (string). If
   not present, locking will be disabled. 
 * `profile` - (Optional) This is the AWS profile name as set in the
   shared credentials file. It can also be sourced from the `AWS_PROFILE`
   environment variable if `AWS_SDK_LOAD_CONFIG` is set to a truthy value,
   e.g. `AWS_SDK_LOAD_CONFIG=1`.
 * `shared_credentials_file`  - (Optional) This is the path to the
   shared credentials file. If this is not set and a profile is specified,
   `~/.aws/credentials` will be used.
 * `token` - (Optional) Use this to set an MFA token. It can also be
   sourced from the `AWS_SESSION_TOKEN` environment variable.
 * `role_arn` - (Optional) The role to be assumed.
 * `assume_role_policy` - (Optional) The permissions applied when assuming a role.
 * `external_id` - (Optional) The external ID to use when assuming the role.
 * `session_name` - (Optional) The session name to use when assuming the role.
 * `workspace_key_prefix` - (Optional) The prefix applied to the state hash key
   inside the table. This is only relevant when using a non-default workspace. This defaults to "workspace"
 * `dynamodb_endpoint` / `AWS_DYNAMODB_ENDPOINT` - (Optional) A custom endpoint for the DynamoDB API.
 * `iam_endpoint` / `AWS_IAM_ENDPOINT` - (Optional) A custom endpoint for the IAM API.
 * `sts_endpoint` / `AWS_STS_ENDPOINT` - (Optional) A custom endpoint for the STS API.
 * `force_path_style` - (Optional) Always use path-style DynamoDB URLs (`https://<HOST>/<table>` instead of `https://<table>.<HOST>`).
 * `skip_credentials_validation` - (Optional) Skip the credentials validation via the STS API.
 * `skip_region_validation` - (Optional) Skip validation of provided region name.
 * `skip_metadata_api_check` - (Optional) Skip the AWS Metadata API check.
 * `max_retries` - (Optional) The maximum number of times an AWS API request is retried on retryable failure. Defaults to 5.


[DynamoDB Encryotion at Rest](https://docs.aws.amazon.com/amazondynamodb/latest/developerguide/EncryptionAtRest.html) is enabled by default.

## Multi-account AWS Architecture

A common architectural pattern is for an organization to use a number of
separate AWS accounts to isolate different teams and environments. For example,
a "staging" system will often be deployed into a separate AWS account than
its corresponding "production" system, to minimize the risk of the staging
environment affecting production infrastructure, whether via rate limiting,
misconfigured access controls, or other unintended interactions.

The DynamoDB backend can be used in a number of different ways that make different
tradeoffs between convenience, security, and isolation in such an organization.
This section describes one such approach that aims to find a good compromise
between these tradeoffs, allowing use of
[Terraform's workspaces feature](/docs/state/workspaces.html) to switch
conveniently between multiple isolated deployments of the same configuration.

Use this section as a starting-point for your approach, but note that
you will probably need to make adjustments for the unique standards and
regulations that apply to your organization. You will also need to make some
adjustments to this approach to account for _existing_ practices within your
organization, if for example other tools have previously been used to manage
infrastructure.

Terraform is an administrative tool that manages your infrastructure, and so
ideally the infrastructure that is used by Terraform should exist outside of
the infrastructure that Terraform manages. This can be achieved by creating a
separate _administrative_ AWS account which contains the user accounts used by
human operators and any infrastructure and tools used to manage the other
accounts. Isolating shared administrative tools from your main environments
has a number of advantages, such as avoiding accidentally damaging the
administrative infrastructure while changing the target infrastructure, and
reducing the risk that an attacker might abuse production infrastructure to
gain access to the (usually more privileged) administrative infrastructure.

### Administrative Account Setup

Your administrative AWS account will contain at least the following items:

* One or more [IAM user](http://docs.aws.amazon.com/IAM/latest/UserGuide/id_users.html)
  for system administrators that will log in to maintain infrastructure in
  the other accounts.
* Optionally, one or more [IAM groups](http://docs.aws.amazon.com/IAM/latest/UserGuide/id_groups.html)
  to differentiate between different groups of users that have different
  levels of access to the other AWS accounts.
* A [DynamoDB table](http://docs.aws.amazon.com/amazondynamodb/latest/developerguide/HowItWorks.CoreComponents.html#HowItWorks.CoreComponents.TablesItemsAttributes)
  that will contain the Terraform state files for each workspace.
* A [DynamoDB table](http://docs.aws.amazon.com/amazondynamodb/latest/developerguide/HowItWorks.CoreComponents.html#HowItWorks.CoreComponents.TablesItemsAttributes)
  that will be used for locking to prevent concurrent operations on a single
  workspace.

Provide the DynamoDB state table name and DynamoDB lock table name to Terraform within the
DynamoDB backend configuration using the `state_table` and `lock_table` arguments
respectively, and configure a suitable `workspace_key_prefix` to contain
the states of the various workspaces that will subsequently be created for
this configuration.

### Environment Account Setup

For the sake of this section, the term "environment account" refers to one
of the accounts whose contents are managed by Terraform, separate from the
administrative account described above.

Your environment accounts will eventually contain your own product-specific
infrastructure. Along with this it must contain one or more
[IAM roles](http://docs.aws.amazon.com/IAM/latest/UserGuide/id_roles.html)
that grant sufficient access for Terraform to perform the desired management
tasks.

### Delegating Access

Each Administrator will run Terraform using credentials for their IAM user
in the administrative account.
[IAM Role Delegation](http://docs.aws.amazon.com/IAM/latest/UserGuide/tutorial_cross-account-with-roles.html)
is used to grant these users access to the roles created in each environment
account.

Full details on role delegation are covered in the AWS documentation linked
above. The most important details are:

* Each role's _Assume Role Policy_ must grant access to the administrative AWS
  account, which creates a trust relationship with the administrative AWS
  account so that its users may assume the role.
* The users or groups within the administrative account must also have a
  policy that creates the converse relationship, allowing these users or groups
  to assume that role.

Since the purpose of the administrative account is only to host tools for
managing other accounts, it is useful to give the administrative accounts
restricted access only to the specific operations needed to assume the
environment account role and access the Terraform state. By blocking all
other access, you remove the risk that user error will lead to staging or
production resources being created in the administrative account by mistake.

When configuring Terraform, use either environment variables or the standard
credentials file `~/.aws/credentials` to provide the administrator user's
IAM credentials within the administrative account to both the DynamoDB backend _and_
to Terraform's AWS provider.

Use conditional configuration to pass a different `assume_role` value to
the AWS provider depending on the selected workspace. For example:

```hcl
variable "workspace_iam_roles" {
  default = {
    staging    = "arn:aws:iam::STAGING-ACCOUNT-ID:role/Terraform"
    production = "arn:aws:iam::PRODUCTION-ACCOUNT-ID:role/Terraform"
  }
}

provider "aws" {
  # No credentials explicitly set here because they come from either the
  # environment or the global credentials file.

  assume_role = "${var.workspace_iam_roles[terraform.workspace]}"
}
```

If workspace IAM roles are centrally managed and shared across many separate
Terraform configurations, the role ARNs could also be obtained via a data
source such as [`terraform_remote_state`](/docs/providers/terraform/d/remote_state.html)
to avoid repeating these values.

### Creating and Selecting Workspaces

With the necessary objects created and the backend configured, run
`terraform init` to initialize the backend and establish an initial workspace
called "default". This workspace will not be used, but is created automatically
by Terraform as a convenience for users who are not using the workspaces
feature.

Create a workspace corresponding to each key given in the `workspace_iam_roles`
variable value above:

```
$ terraform workspace new staging
Created and switched to workspace "staging"!

...

$ terraform workspace new production
Created and switched to workspace "production"!

...
```

Due to the `assume_role` setting in the AWS provider configuration, any
management operations for AWS resources will be performed via the configured
role in the appropriate environment AWS account. The backend operations, such
as reading and writing the state from DynamoDB, will be performed directly as the
administrator's own user within the administrative account.

```
$ terraform workspace select staging
$ terraform apply
...
```

### Running Terraform in Amazon EC2

Teams that make extensive use of Terraform for infrastructure management
often [run Terraform in automation](/guides/running-terraform-in-automation.html)
to ensure a consistent operating environment and to limit access to the
various secrets and other sensitive information that Terraform configurations
tend to require.

When running Terraform in an automation tool running on an Amazon EC2 instance,
consider running this instance in the administrative account and using an
[instance profile](http://docs.aws.amazon.com/IAM/latest/UserGuide/id_roles_use_switch-role-ec2_instance-profiles.html)
in place of the various administrator IAM users suggested above. An IAM
instance profile can also be granted cross-account delegation access via
an IAM policy, giving this instance the access it needs to run Terraform.

To isolate access to different environment accounts, use a separate EC2
instance for each target account so that its access can be limited only to
the single account.

Similar approaches can be taken with equivalent features in other AWS compute
services, such as ECS.

### Protecting Access to Workspace State

In a simple implementation of the pattern described in the prior sections,
all users have access to read and write states for all workspaces. In many
cases it is desirable to apply more precise access constraints to the
Terraform state objects in DynamoDB, so that for example only trusted administrators
are allowed to modify the production state, or to control _reading_ of a state
that contains sensitive information.

Amazon DynamoDB supports fine-grained access control on a per-object-path basis
using IAM policy. A full description of DynamoDB's access control mechanism is
beyond the scope of this guide, but an example IAM policy granting access
to only a single state object within an DynamoDB table is shown below:

```json
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Effect": "Allow",
      "Action": "DynamoDB:Listtable",
      "Resource": "arn:aws:DynamoDB:::myorg-terraform-states"
    },
    {
      "Effect": "Allow",
      "Action": ["DynamoDB:GetObject", "DynamoDB:PutObject"],
      "Resource": "arn:aws:DynamoDB:::myorg-terraform-states/myapp/production/tfstate"
    }
  ]
}
```

It is not possible to apply such fine-grained access control to the DynamoDB
table used for locking, so it is possible for any user with Terraform access
to lock any workspace state, even if they do not have access to read or write
that state. If a malicious user has such access they could block attempts to
use Terraform against some or all of your workspaces as long as locking is
enabled in the backend configuration.
