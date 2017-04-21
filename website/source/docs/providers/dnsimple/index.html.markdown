---
layout: "dnsimple"
page_title: "Provider: DNSimple"
sidebar_current: "docs-dnsimple-index"
description: |-
  The DNSimple provider is used to interact with the resources supported by DNSimple. The provider needs to be configured with the proper credentials before it can be used.
---

# DNSimple Provider

The DNSimple provider is used to interact with the
resources supported by DNSimple. The provider needs to be configured
with the proper credentials before it can be used.

Use the navigation to the left to read about the available resources.


## Example Usage

```hcl
# Configure the DNSimple provider
provider "dnsimple" {
  token = "${var.dnsimple_token}"
  account = "${var.dnsimple_account}"
}

# Create a record
resource "dnsimple_record" "www" {
  # ...
}
```


## API v2 vs API v1

This integration uses the new DNSimple API v2 [released on December 2016](https://blog.dnsimple.com/2016/12/api-v2-stable/). The API v2 provides support for multi-accounts and requires a new authentication mechanism.

If you are upgrading from a previous Terraform version and you were using the API v1, you will need to upgrade the DNSimple provider configuration to use the new API access token and specify the Account ID. Terraform will automatically detect an existing legacy configurations and it will return an error message asking to upgrade.

API v1 is no longer supported. If you are using the `DNSIMPLE_EMAIL` argument, you can safely remove it once you have upgraded to API v2. To use API v1 you will need to use a Terraform version lower than 0.9.

To upgrade from the DNSimple provider API v1 to DNSimple provider API v2 follow these steps:

1. [Generate an API v2 access token](https://support.dnsimple.com/articles/api-access-token/)
1. [Determine the Account ID](https://developer.dnsimple.com/v2/#account-scope)
1. Add the `account` configuration and update the `token`, as shown in the example above
1. Remove the `email` configuration, as it's no longer used


## Argument Reference

The following arguments are supported:

* `token` - (Required) The DNSimple API v2 token. It must be provided, but it can also be sourced from the `DNSIMPLE_TOKEN` environment variable. Please note that this must be an [API v2 token](https://support.dnsimple.com/articles/api-access-token/). You can use either an User or Account token, but an Account token is recommended.
* `account` - (Required) The ID of the account associated with the token. It must be provided, but it can also be sourced from the `DNSIMPLE_ACCOUNT` environment variable.
