---
layout: "puppetdb"
page_title: "Provider: PuppetDB"
sidebar_current: "docs-puppetdb-index"
description: |-
  The PuppetDB provider is used to interact with the PuppetDB.
---

# PuppetDB Provider

The PuppetDB provider is used to interact with the
resources supported by PuppetDB. The provider needs to be configured
with the URL of the PuppetDB server at minimum and SSL keys if using HTTPS.

## Example Usage

```hcl
# Configure the PuppetDB provider
provider "puppetdb" {
  url        = "https://puppetdb.my-domain.com:8081"
  key        = "certs/my.key"
  cert       = "certs/my.crt"
  ca         = "certs/ca.crt"
}
```

## Argument Reference

The following arguments are supported:

* `url` - (Required) PuppetDB url. It must be provided, but it can also be sourced from the `PUPPETDB_URL` environment variable.
* `key` - (Optional) SSL key used for authentication. It can also be sourced from the `PUPPETDB_KEY` environment variable.
* `cert` - (Optional) SSL certificate used for authentication. It can also be sourced from the `PUPPETDB_CERT` environment variable.
* `ca` - (Optional) SSL CA certificate used for authentication. It can also be sourced from the `PUPPETDB_CA` environment variable.
