---
layout: "namecheap"
page_title: "Provider: Namecheap"
sidebar_current: "docs-namecheap-index"
description: |-
  The Namecheap provider is used to interact with the resources supported by Namecheap. The provider needs to be configured with the proper credentials before it can be used.
---

# Namecheap Provider

The Namecheap provider is used to interact with the
resources supported by Namecheap. The provider needs to be configured
with the proper credentials before it can be used.

Use the navigation to the left to read about the available resources.

## Example Usage

```
# Configure the Namecheap provider
provider "namecheap" {
    username = "${var.namecheap_username}"
    api_user = "${var.namecheap_apiuser}"
    token = "${var.namecheap_token}"
    ip = "${var.namecheap_ip}"
    use_sandbox = true

# Create a record
resource "namecheap_record" "www" {
    ...
}
```

## Argument Reference

The following arguments are supported:

* `username` - (Required) The Namecheap account username. It must be provided, but it can also be sourced from the `NAMECHEAP_USERNAME` environment variable.
* `api_user` - (Required) The Namecheap apiuser. It must be provided, but it can also be sourced from the `NAMECHEAP_API_USER` environment variable.
* `token` - (Required) The Namecheap API token. It must be provided, but it can also be sourced from the `NAMECHEAP_TOKEN` environment variable.
* `ip` - (Required) Whitelisted Namecheap ip. It must be provided, but it can also be sourced from the `NAMECHEAP_IP` environment variable.
* `use_sandbox` - (Required) Determines if the sandbox api is used or not. It must be provided, but it can also be sourced from the `NAMECHEAP_USE_SANDBOX` environment variable.


