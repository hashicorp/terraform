---
layout: "aws"
page_title: "AWS: aws_iam_openid_connect_provider"
sidebar_current: "docs-aws-resource-iam-openid-connect-provider"
description: |-
  Provides an IAM OpenID Connect provider.
---

# aws\_iam\_openid\_connect\_provider

Provides an IAM OpenID Connect provider.

## Example Usage

```hcl
resource "aws_iam_openid_connect_provider" "default" {
    url = "https://accounts.google.com"
    client_id_list = [
     "266362248691-342342xasdasdasda-apps.googleusercontent.com"
    ]
    thumbprint_list = []
}
```

## Argument Reference

The following arguments are supported:

* `url` - (Required) The URL of the identity provider. Corresponds to the _iss_ claim.
* `client_id_list` - (Required) A list of client IDs (also known as audiences). When a mobile or web app registers with an OpenID Connect provider, they establish a value that identifies the application. (This is the value that's sent as the client_id parameter on OAuth requests.)
* `thumbprint_list` - (Required) A list of server certificate thumbprints for the OpenID Connect (OIDC) identity provider's server certificate(s). 

## Attributes Reference

The following attributes are exported:

* `arn` - The ARN assigned by AWS for this provider.

## Import

IAM OpenID Connect Providers can be imported using the `arn`, e.g.

```
$ terraform import aws_iam_openid_connect_provider.default arn:aws:iam::123456789012:oidc-provider/accounts.google.com
```
