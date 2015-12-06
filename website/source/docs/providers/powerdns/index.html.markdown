---
layout: "powerdns"
page_title: "Provider: PowerDNS"
sidebar_current: "docs-powerdns-index"
description: |-
  The PowerDNS provider is used manipulate DNS records supported by PowerDNS server. The provider needs to be configured with the proper credentials before it can be used.
---

# PowerDNS Provider

The PowerDNS provider is used manipulate DNS records supported by PowerDNS server. The provider needs to be configured
with the proper credentials before it can be used.

Use the navigation to the left to read about the available resources.

## Example Usage

```
# Configure the PowerDNS provider
provider "powerdns" {
    api_key = "${var.pdns_api_key}"
    server_url = "${var.pdns_server_url}"
}

# Create a record
resource "powerdns_record" "www" {
    ...
}
```

## Argument Reference

The following arguments are supported:

* `api_key` - (Required) The PowerDNS API key. This can also be specified with `PDNS_API_KEY` environment variable.
* `server_url` - (Required) The address of PowerDNS server. This can also be specified with `PDNS_SERVER_URL` environment variable.
