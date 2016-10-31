---
description: 'Provides an NS1 zone resource.'
layout: nsone
page_title: 'NS1: ns1_zone'
sidebar_current: 'docs-nsone-resource-zone'
---

# ns1\_zone

Provides an NS1 zone resource.

## Example Usage

    # Add a minimal zone
    resource "ns1_zone" "foobar" {
      zone = "${var.ns1_domain}"
    }

    # Add a complete zome
    resource "ns1_zone" "foobar" {
      zone        = "${var.ns1_domain}"
      ttl         = 10800
      refresh     = 3600
      retry       = 300
      expiry      = 2592000
      nx_ttl      = 1234
    }

## Argument Reference

See [the NS1 API Docs](https://ns1.com/api/) for details about valid
values. Many are originally defined in
[RFC-1035](https://tools.ietf.org/html/rfc1035).

The following arguments are supported:

-   `zone` - (Required) - The domain of the zone
-   `ttl` - (Optional) The TTL of the zone
-   `refresh` - `REFRESH`, "A 32 bit time interval before the zone
    should be refreshed." per RFC1035
-   `retry` - `RETRY`, "A 32 bit time interval that should elapse before
    a failed refresh should be retried." per RFC1035
-   `expiry` - `EXPIRE`, "A 32 bit time value that specifies the upper
    limit on the time interval that can elapse before the zone is no
    longer authoritative." per RFC1035
-   `nx_ttl` - aka, the SOA MINIMUM field, overloaded for NXDOMAIN per
    [RFC2308](https://tools.ietf.org/html/rfc2308). `MINIMUM`, "The
    unsigned 32 bit minimum TTL field that should be exported with any
    RR from this zone." per RFC1035
-   `link` - zone to mirror
-   `primary` - AXFR server to mirror

## Attributes Reference

The following attributes are exported:

-   `id` - The zone ID
-   `dns_servers` - used to provide SOA `MNAME`, "The <domain-name> of
    the name server that was the original or primary source of data for
    this zone." per RFC1035
-   `hostmaster` - aka `RNAME`, "A <domain-name> which specifies the
    mailbox of the person responsible for this zone." per RFC1035
