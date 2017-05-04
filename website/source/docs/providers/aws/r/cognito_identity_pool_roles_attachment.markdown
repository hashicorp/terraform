---
layout: "aws"
page_title: "AWS: aws_cognito_identity_pool_roles_attachment"
sidebar_current: "docs-aws-resource-cognito-identity-pool-roles-attachment"
description: |-
  Provides an AWS Cognito Identity Pool Roles Attachment.
---

# aws\_cognito\_identity\_pool\_roles\_attachment

Provides an AWS Cognito Identity Pool Roles Attachment.

## Example Usage

```
resource "aws_cognito_identity_pool" "main" {
  identity_pool_name               = "identity pool"
  allow_unauthenticated_identities = false

  supported_login_providers {
    "graph.facebook.com" = "7346241598935555"
  }
}

resource "aws_iam_role" "authenticated" {
  name = "cognito_authenticated"

  assume_role_policy = <<EOF
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Effect": "Allow",
      "Principal": {
        "Federated": "cognito-identity.amazonaws.com"
      },
      "Action": "sts:AssumeRoleWithWebIdentity",
      "Condition": {
        "StringEquals": {
          "cognito-identity.amazonaws.com:aud": "${aws_cognito_identity_pool.main.id}"
        },
        "ForAnyValue:StringLike": {
          "cognito-identity.amazonaws.com:amr": "authenticated"
        }
      }
    }
  ]
}
EOF
}

resource "aws_iam_role_policy" "authenticated" {
  name = "authenticated_policy"
  role = "${aws_iam_role.authenticated.id}"

  policy = <<EOF
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Effect": "Allow",
      "Action": [
        "mobileanalytics:PutEvents",
        "cognito-sync:*",
        "cognito-identity:*"
      ],
      "Resource": [
        "*"
      ]
    }
  ]
}
EOF
}

resource "aws_cognito_identity_pool_roles_attachment" "main" {
  identity_pool_id = "${aws_cognito_identity_pool.main.id}"

  role_mappings {
    role_mapping {
      key   = "graph.facebook.com"
      value = {
        ambiguous_role_resolution = "AuthenticatedRole"
        type                      = "Rules"

        rules_configuration {
          rules {
            claim      = "isAdmin"
            match_type = "Equals"
            role_arn   = "${aws_iam_role.authenticated.arn}"
            value      = "paid"
          }
        }
      }
    }
  }

  roles {
    "authenticated" = "${aws_iam_role.authenticated.arn}"
  }
}
```

## Argument Reference

The Cognito Identity Pool Roles Attachment argument layout is a structure composed of several sub-resources - these resources are laid out below.

* `identity_pool_id` (Required) - An identity pool ID in the format REGION:GUID.
* `role_mappings` (Optional) - A List of [Role Mapping](#role-mappings).
* `roles` (Required) - The map of roles associated with this pool. For a given role, the key will be either "authenticated" or "unauthenticated" and the value will be the Role ARN.

#### Role Mappings

* `role_mapping` (Optional) - A set of [Role Mapping entries](#role-mapping-entries).

#### Role Mapping Entries

* `key` (Required) - A string identifying the identity provider, for example, "graph.facebook.com" or "cognito-idp-east-1.amazonaws.com/us-east-1_abcdefghi:app_client_id".
* `value` (Required) - A [Role Mapping](#role-mapping) structure.

#### Role Mapping

* `ambiguous_role_resolution` (Optional) - Specifies the action to be taken if either no rules match the claim value for the Rules type, or there is no cognito:preferred_role claim and there are multiple cognito:roles matches for the Token type. `Required` if you specify Token or Rules as the Type.
* `rules_configuration` (Optional) - The [Rules Configuration](#rules-configuration) to be used for mapping users to roles.
* `type` (Required) - The role mapping type.

#### Rules Configuration

* `rules` (Required) - An array of [rules](#rules). You can specify up to 25 rules per identity provider. Rules are evaluated in order. The first one to match specifies the role.

#### Rules

* `claim` (Required) - The claim name that must be present in the token, for example, "isAdmin" or "paid".
* `match_type` (Required) - The match condition that specifies how closely the claim value in the IdP token must match Value.
* `role_arn` (Required) - The role ARN.
* `value` (Required) - A brief string that the claim must match, for example, "paid" or "yes".

## Attributes Reference

In addition to the arguments, which are exported, the following attributes are exported:

* `id` - The identity pool ID.
* `identity_pool_id` (Required) - An identity pool ID in the format REGION:GUID.
* `role_mappings` (Optional) - The List of [Role Mapping](#role-mappings).
* `roles` (Required) - The map of roles associated with this pool. For a given role, the key will be either "authenticated" or "unauthenticated" and the value will be the Role ARN.
