---
layout: "dme"
page_title: "Provider: DNSMadeEasy"
sidebar_current: "docs-dme-index"
description: |-
  The DNSMadeEasy provider is used to interact with the resources supported by DNSMadeEasy. The provider needs to be configured with the proper credentials before it can be used.
---

# DNSMadeEasy Provider

The DNSMadeEasy provider is used to interact with the
resources supported by DNSMadeEasy. The provider needs to be configured
with the proper credentials before it can be used.

Use the navigation to the left to read about the available resources.

## Example Usage

```hcl
# Configure the DNSMadeEasy provider
provider "dme" {
  akey       = "${var.dme_akey}"
  skey       = "${var.dme_skey}"
  usesandbox = true
}

# Create an A record
resource "dme_record" "www" {
  domainid = "123456"

  # ...
}
```

## Argument Reference

The following arguments are supported:

* `akey` - (Required) The DNSMadeEasy API key. This can also be specified with
  the `DME_AKEY` shell environment variable.
* `skey` - (Required) The DNSMadeEasy Secret key. This can also be specified
  with the `DME_SKEY` shell environment variable.
* `usesandbox` - (Optional) If true, the DNSMadeEasy sandbox will be
  used. This can also be specified with the `DME_USESANDBOX` shell environment
  variable.
