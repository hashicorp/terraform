---
layout: "aws"
page_title: "AWS: aws_cognito_identity_pool"
sidebar_current: "docs-aws-resource-cognito-identity-pool"
description: |-
  Provides an AWS Cognito Identity Pool.
---

# aws\_cognito\_identity\_pool

Provides an AWS Cognito Identity Pool.

## Example Usage

```
resource "aws_iam_saml_provider" "default" {
  name                   = "my-saml-provider"
  saml_metadata_document = "${file("saml-metadata.xml")}"
}

resource "aws_cognito_identity_pool" "main" {
  identity_pool_name               = "identity pool"
  allow_unauthenticated_identities = false

  cognito_identity_providers {
    client_id               = "6lhlkkfbfb4q5kpp90urffae"
    provider_name           = "cognito-idp.us-east-1.amazonaws.com/us-east-1_Tv0493apJ"
    server_side_token_check = false
  }

  cognito_identity_providers {
    client_id               = "7kodkvfqfb4qfkp39eurffae"
    provider_name           = "cognito-idp.us-east-1.amazonaws.com/eu-west-1_Zr231apJu"
    server_side_token_check = false
  }

  supported_login_providers {
    "graph.facebook.com"  = "7346241598935552"
    "accounts.google.com" = "123456789012.apps.googleusercontent.com"
  }

  saml_provider_arns           = ["${aws_iam_saml_provider.default.arn}"]
  openid_connect_provider_arns = ["arn:aws:iam::123456789012:oidc-provider/foo.example.com"]
}
```

## Argument Reference

The Cognito Identity Pool argument layout is a structure composed of several sub-resources - these resources are laid out below.

* `identity_pool_name` (Required) - The Cognito Identity Pool name.
* `allow_unauthenticated_identities` (Required) - Whether the identity pool supports unauthenticated logins or not.
* `developer_provider_name` (Optional) - The "domain" by which Cognito will refer to your users. This name acts as a placeholder that allows your
backend and the Cognito service to communicate about the developer provider.
* `cognito_identity_providers` (Optional) - An array of [Amazon Cognito Identity user pools](#cognito-identity-providers) and their client IDs.
* `openid_connect_provider_arns` (Optional) - A list of OpendID Connect provider ARNs.
* `saml_provider_arns` (Optional) - An array of Amazon Resource Names (ARNs) of the SAML provider for your identity.
* `supported_login_providers` (Optional) - Key-Value pairs mapping provider names to provider app IDs.

#### Cognito Identity Providers

  * `client_id` (Optional) - The client ID for the Amazon Cognito Identity User Pool.
  * `provider_name` (Optional) - The provider name for an Amazon Cognito Identity User Pool.
  * `server_side_token_check` (Optional) - Whether server-side token validation is enabled for the identity provider’s token or not.

## Attributes Reference

In addition to the arguments, which are exported, the following attributes are exported:

* `id` - An identity pool ID in the format REGION:GUID.

## Import

Cognito Identity Pool can be imported using the name, e.g.

```
$ terraform import aws_cognito_identity_pool.mypool <identity-pool-id>
```
