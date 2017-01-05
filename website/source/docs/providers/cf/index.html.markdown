---
layout: "cf"
page_title: "Provider: Cloud Foundry"
sidebar_current: "docs-cf-index"
description: |-
  The Cloud Foundry (cloudfoundry) provider is used to manage a Cloud Foundry environment. The provider needs to be configured with the proper credentials before it can be used.
---

# Cloud Foundry Provider

The Cloud Foundry (cloudfoundry) provider is used to interact with a
Cloud Foundry target to perform adminstrative configuration of platform 
resources.

Use the navigation to the left to read about the available resources.

## Example Usage

```
# Set the variable values in *.tfvars file
# or using -var="api_url=..." CLI option

variable "api_url" {}
variable "admin_password" {}
variable "uaa_admin_client_secret" {}

# Configure the CloudFoundry Provider

provider "cloudfoundry" {
    api_url = "${var.api_url}"
    user = "admin"
    password = "${var.admin_password}"
    uaa_client_id = "admin"
    uaa_client_secret = "${var.uaa_admin_client_secret}"
    skip_ssl_validation = true
}
```

## Argument Reference

The following arguments are supported:

* `api_url` - (Required) API endpoint (e.g. https://api.local.pcfdev.io). This can also be specified
  with the `CF_API_URL` shell environment variable.

* `user` - (Optional) Cloud Foundry user wih admin privileges. Defaults to "admin". This can also be specified
  with the `CF_USER` shell environment variable.

* `password` - (Required) Cloud Foundry admin user's password. This can also be specified
  with the `CF_PASSWORD` shell environment variable.

* `uaa_client_id` - (Optional) The UAA admin client ID. Defaults to "admin". This can also be specified
  with the `CF_UAA_CLIENT_ID` shell environment variable.

* `uaa_client_secret` - (Required) This secret of the UAA admin client. This can also be specified
  with the `CF_UAA_CLIENT_SECRET` shell environment variable.

* `skip_ssl_validation` - (Optional) Skip verification of the API endpoint - Not recommended!. Defaults to "false". This can also be specified
  with the `CF_SKIP_SSL_VALIDATION` shell environment variable.
