---
layout: "dnsimple"
page_title: "Provider: DNSimple"
sidebar_current: "docs-dnsimple-index"
---

# DNSimple Provider

The DNSimple provider is used to interact with the
resources supported by DNSimple. The provider needs to be configured
with the proper credentials before it can be used.

Use the navigation to the left to read about the available resources.

## Example Usage

```
# Configure the DNSimple provider
provider "dnsimple" {
    token = "${var.dnsimple_token}"
    email = "${var.dnsimple_email}"
}

# Create a record
resource "dnsimple_record" "www" {
    ...
}
```

## Argument Reference

The following arguments are supported:

* `token` - (Required) The DNSimple API token
* `email` - (Required) The email associated with the token


