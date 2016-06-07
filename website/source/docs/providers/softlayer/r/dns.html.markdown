---
layout: "softlayer"
page_title: "SoftLayer: softlayer_dns_domain"
sidebar_current: "docs-softlayer-resource-softlayer-dns-domain"
description: |-
  Provides a Softlayer's DNS Domain.
---

# softlayer_dns_domain

The `softLayer_dns_domain` data type represents a single DNS domain record hosted on the SoftLayer nameservers. Domains contain general information about the domain name such as name and serial. Individual records such as `A`, `AAAA`, `CTYPE`, and `MX` records are stored in the domain's associated resource records using the  [`softlayer_dns_domain_resourcerecord`](/docs/providers/softlayer/r/dns_records.html) resource.

## Example Usage

```
resource "softlayer_dns_domain" "dns-domain-test" {
	name = "dns-domain-test.com"
}
```


## Argument Reference
The following arguments are supported:

* `name` | *string*
     * (Required) A domain's name including top-level domain, for example "example.com". _Name_ is the only field that needs to be set for `softlayer_dns_domain`. During creation the `NS` and `SOA` resource records are created automatically.

## Attributes Reference
The following attributes are exported

* `id` - A domain record's internal identifier.
* `serial` - A unique number denoting the latest revision of a domain.
* `update_date` - The date that this domain record was last updated.