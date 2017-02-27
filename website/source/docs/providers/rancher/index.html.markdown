---
layout: "rancher"
page_title: "Provider: Rancher"
sidebar_current: "docs-rancher-index"
description: |-
  The Rancher provider is used to interact with Rancher container platforms.
---

# Rancher Provider

The Rancher provider is used to interact with the
resources supported by Rancher. The provider needs to be configured
with the URL of the Rancher server at minimum and API credentials if
access control is enabled on the server.

## Example Usage

```hcl
# Configure the Rancher provider
provider "rancher" {
  api_url    = "http://rancher.my-domain.com:8080"
  access_key = "${var.rancher_access_key}"
  secret_key = "${var.rancher_secret_key}"
}
```

## Argument Reference

The following arguments are supported:

* `api_url` - (Required) Rancher API url. It must be provided, but it can also be sourced from the `RANCHER_URL` environment variable.
* `access_key` - (Optional) Rancher API access key. It can also be sourced from the `RANCHER_ACCESS_KEY` environment variable.
* `secret_key` - (Optional) Rancher API access key. It can also be sourced from the `RANCHER_SECRET_KEY` environment variable.
