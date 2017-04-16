---
layout: "dns"
page_title: "Provider: DNS"
sidebar_current: "docs-dns-index"
description: |-
  The DNS provider supports DNS updates (RFC 2136). Additionally, the provider can be configured with secret key based transaction authentication (RFC 2845).
---

# DNS Provider

The DNS provider supports DNS updates (RFC 2136). Additionally, the provider can be configured with secret key based transaction authentication (RFC 2845).

Use the navigation to the left to read about the available resources.

## Example Usage

```hcl
# Configure the DNS Provider
provider "dns" {
  update {
    server        = "192.168.0.1"
    key_name      = "example.com."
    key_algorithm = "hmac-md5"
    key_secret    = "3VwZXJzZWNyZXQ="
  }
}

# Create a DNS A record set
resource "dns_a_record_set" "www" {
  # ...
}
```

## Configuration Reference

`update` - (Optional) When the provider is used for DNS updates, this block is required. Structure is documented below.

The `update` block supports the following attributes:

* `server` - (Required) The IPv4 address of the DNS server to send updates to.
* `port` - (Optional) The target UDP port on the server where updates are sent to. Defaults to `53`.
* `key_name` - (Optional) The name of the TSIG key used to sign the DNS update messages.
* `key_algorithm` - (Optional; Required if `key_name` is set) When using TSIG authentication, the algorithm to use for HMAC. Valid values are `hmac-md5`, `hmac-sha1`, `hmac-sha256` or `hmac-sha512`.
* `key_secret` - (Optional; Required if `key_name` is set)
    A Base64-encoded string containing the shared secret to be used for TSIG.
