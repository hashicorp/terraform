---
layout: "softlayer"
page_title: "SoftLayer: softlayer_dns_domain_resourcerecord"
sidebar_current: "docs-softlayer-resource-softlayer-dns-domain-resourcerecord"
description: |-
  Provides a Softlayer's DNS Domain Records
---

# softlayer_dns_domain_resourcerecord

The `softlayer_dns_domain_resourcerecord` data type represents a single resource record entry in a [`softlayer_dns_domain`](/docs/providers/softlayer/r/dns.html). Each resource record contains a `host` and `record_data` property, defining a resource's name and it's target data.
## Example Usage

We are using [SoftLayer_Dns_Domain_ResourceRecord](https://sldn.softlayer.com/reference/datatypes/SoftLayer_Dns_Domain_ResourceRecord)
SL's object for most of CRUD operations. Only for SRV record type we are using [SoftLayer_Dns_Domain_ResourceRecord_SrvType](https://sldn.softlayer.com/reference/services/SoftLayer_Dns_Domain_ResourceRecord_SrvType) SL's object.

Currently we can CRUD almost all record types except _SOA_ type which is initially created on DNS create action. 

### Create example:
#### `A` Record | [SLDN](http://sldn.softlayer.com/reference/datatypes/SoftLayer_Dns_Domain_ResourceRecord_AType)
```
resource "softlayer_dns_domain" "main" {
	name = "main.example.com"
}

resource "softlayer_dns_domain_record" "www" {
    record_data = "123.123.123.123"
    domain_id = "${softlayer_dns_domain.main.id}"
    host = "www.example.com"
    contact_email = "user@softlayer.com"
    ttl = 900
    record_type = "a"
}
```
#### `AAAA` Record | [SLDN](http://sldn.softlayer.com/reference/datatypes/SoftLayer_Dns_Domain_ResourceRecord_AaaaType)
```
resource "softlayer_dns_domain_record" "aaaa" {
    record_data = "FE80:0000:0000:0000:0202:B3FF:FE1E:8329"
    domain_id = "${softlayer_dns_domain.main.id}"
    host = "www.example.com"
    contact_email = "user@softlayer.com"
    ttl = 1000
    record_type = "aaaa"
}
```
#### `CNAME` Record | [SLDN](http://sldn.softlayer.com/reference/datatypes/SoftLayer_Dns_Domain_ResourceRecord_CnameType)
```
resource "softlayer_dns_domain_record" "cname" {
    record_data = "real-host.example.com."
    domain_id = "${softlayer_dns_domain.main.id}"
    host = "alias.example.com"
    contact_email = "user@softlayer.com"
    ttl = 900
    record_type = "cname"
}
```
#### `MX` Record | [SLDN](http://sldn.softlayer.com/reference/datatypes/SoftLayer_Dns_Domain_ResourceRecord_MxType)
```
resource "softlayer_dns_domain_record" "recordMX-1" {
    record_data = "mail-1"
    domain_id = "${softlayer_dns_domain.main.id}"
    host = "@"
    mx_priority = "10"
    contact_email = "user@softlayer.com"
    ttl = 900
    record_type = "mx"
}
```
#### `NS` Record | [SLDN](http://sldn.softlayer.com/reference/datatypes/SoftLayer_Dns_Domain_ResourceRecord_NsType)
```
resource "softlayer_dns_domain_record" "recordNS" {
    record_data = "ns1.example.org"
    domain_id = "${softlayer_dns_domain.main.id}"
    host = "@"
    contact_email = "user@softlayer.com"
    ttl = 900
    record_type = "ns"
}
```
#### `SPF` Record | [SLDN](http://sldn.softlayer.com/reference/datatypes/SoftLayer_Dns_Domain_ResourceRecord_SpfType)
```
resource "softlayer_dns_domain_record" "recordSPF" {
    record_data = "v=spf1 mx:mail.example.org ~all"
    domain_id = "${softlayer_dns_domain.main.id}"
    host = "mail-1"
    contact_email = "user@softlayer.com"
    ttl = 900
    record_type = "spf"
}  
```
#### `TXT` Record | [SLDN](http://sldn.softlayer.com/reference/datatypes/SoftLayer_Dns_Domain_ResourceRecord_TxtType/)
```
resource "softlayer_dns_domain_record" "recordTXT" {
    record_data = "host"
    domain_id = "${softlayer_dns_domain.main.id}"
    host = "A SPF test host"
    contact_email = "user@softlayer.com"
    ttl = 900
    record_type = "txt"
}
```
#### `SRV` Record | [SLDN](http://sldn.softlayer.com/reference/datatypes/SoftLayer_Dns_Domain_ResourceRecord_SrvType)
```
resource "softlayer_dns_domain_record" "recordSRV" {
    record_data = "ns1.example.org"
    domain_id = "${softlayer_dns_domain.main.id}"
    host = "hosta-srv.com"
    contact_email = "user@softlayer.com"
    ttl = 900
    record_type = "srv"
	port = 8080
	priority = 3
	protocol = "_tcp"
	weight = 3
	service = "_mail"
}
```

#### `PTR` Record
#####  _A note on creating `PTR` records:_ 

There are a lot of things that make the `PTR` record work properly, please review the [SLDN documentation](http://sldn.softlayer.com/reference/datatypes/SoftLayer_Dns_Domain_ResourceRecord_PtrType/) regarding how they are to be implemented.

```
resource "softlayer_dns_domain_record" "recordPTR" {
    record_data = "ptr.example.com"
    domain_id = "${softlayer_dns_domain.main.id}"
    host = "45"  # <- this is the last octet of IPAddress in the range of the subnet
    contact_email = "user@softlayer.com"
    ttl = 900
    record_type = "ptr"
}
```

## Argument Reference

* `record_data` | *string*
    * (Required) The value of a domain's resource record. This can be an IP address or a hostname. Fully qualified host and domain name data must end with the "." character.
* `domain_id` | *int*
    * (Required) An identifier belonging to the domain that a resource record is associated with.
* `expire` | *int*
    * The amount of time in seconds that a secondary name server (or servers) will hold a zone before it is no longer considered authoritative.
* `host` | *string*
    * (Required) The host defined by a resource record. A value of `"@"` denotes a wildcard.
* `minimum_ttl` | *int*
    * The amount of time in seconds that a domain's resource records are valid. This is also known as a minimum TTL, and can be overridden by an individual resource record's TTL.
* `mx_priority` | *int*
    * Useful in cases where a domain has more than one mail exchanger, the priority property is the priority of the MTA that delivers mail for a domain. A lower number denotes a higher priority, and mail will attempt to deliver through that MTA before moving to lower priority mail servers. Priority is defaulted to 10 upon resource record creation.
* `refresh` | *int* 
    * The amount of time in seconds that a secondary name server should wait to check for a new copy of a DNS zone from the domain's primary name server. If a zone file has changed then the secondary DNS server will update it's copy of the zone to match the primary DNS server's zone.
* `contact_email` | *string* 
    * (Required) The email address of the person responsible for a domain, with the "@" replaced with a `.`. For instance, if root@example.org is responsible for example.org, then example.org's SOA responsibility is `root.example.org.`.
* `retry` | *int* 
    * The amount of time in seconds that a domain's primary name server (or servers) should wait if an attempt to refresh by a secondary name server failed before attempting to refresh a domain's zone with that secondary name server again.
* `ttl` | *int* 
    * (Required) The Time To Live value of a resource record, measured in seconds. TTL is used by a name server to determine how long to cache a resource record. An SOA record's TTL value defines the domain's overall TTL.
* `record_type` | *string* - (Required) A domain resource record's type, valid types are:
    * `a` for address records
    * `aaaa` for address records
    * `cname` for canonical name records
    * `mx` for mail exchanger records
    * `ns` for name server records
    * `ptr` for pointer records in reverse domains
    * `soa` for a domain's start of authority record
    * `spf` for sender policy framework records
    * `srv` for service records
* `txt` | *string* 
    * for text records
* `service` | *string* 
    * The symbolic name of the desired service
* `protocol` | *string* 
    * The protocol of the desired service; this is usually either TCP or UDP.
* `port` | *int* 
    * The TCP or UDP port on which the service is to be found.
* `priority` | *int* 
    * The priority of the target host, lower value means more preferred.
* `weight` | *int* 
    * A relative weight for records with the same priority.

## Attributes Reference

* `id` - A domain resource record's internal identifier.