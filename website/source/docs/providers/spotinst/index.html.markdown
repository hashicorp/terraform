---
layout: "spotinst"
page_title: "Provider: Spotinst"
sidebar_current: "docs-spotinst-index"
description: |-
  The Spotinst provider is used to interact with the resources supported by Spotinst. The provider needs to be configured with the proper credentials before it can be used.
---

# Spotinst Provider

The Spotinst provider is used to interact with the
resources supported by Spotinst. The provider needs to be configured
with the proper credentials before it can be used.

Use the navigation to the left to read about the available resources.

## Example Usage

```hcl
# Configure the Spotinst provider
provider "spotinst" {
    email         = "${var.spotinst_email}"
    password      = "${var.spotinst_password}"
    client_id     = "${var.spotinst_client_id}"
    client_secret = "${var.spotinst_client_secret}"
    token         = "${var.spotinst_token}"
}

# Create an AWS group
resource "spotinst_aws_group" "foo" {
    ...
}
```

## Argument Reference

The following arguments are supported:

* `email` - (Required) The email registered in Spotinst. It must be provided, but it can also be sourced from the `SPOTINST_EMAIL` environment variable.
* `password` - (Optional; Required if not using `token`) The password associated with the username. It can be sourced from the `SPOTINST_PASSWORD` environment variable.
* `client_id` - (Optional; Required if not using `token`) The OAuth client ID associated with the username. It can be sourced from the `SPOTINST_CLIENT_ID` environment variable.
* `client_secret` - (Optional; Required if not using `token`) The OAuth client secret associated with the username. It can be sourced from the `SPOTINST_CLIENT_SECRET` environment variable.
* `token` - (Optional; Required if not using `password`) A Personal API Access Token issued by Spotinst. It can be sourced from the `SPOTINST_TOKEN` environment variable.
