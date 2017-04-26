---
layout: "dme"
page_title: "DNSMadeEasy: dme_record"
sidebar_current: "docs-dme-resource-record"
description: |-
  Provides a DNSMadeEasy record resource.
---

# dme_record

Provides a DNSMadeEasy record resource.

## Example Usage

```hcl
# Add an A record to the domain
resource "dme_record" "www" {
  domainid    = "123456"
  name        = "www"
  type        = "A"
  value       = "192.168.1.1"
  ttl         = 3600
  gtdLocation = "DEFAULT"
}
```

## Argument Reference

The following arguments are supported:

* `domainid` - (String, Required) The domain id to add the
  record to
* `name` - (Required) The name of the record `type` - (Required) The type of
* the record `value` - (Required) The value of the record; its usage
  will depend on the `type` (see below)
* `ttl` - (Integer, Optional) The TTL of the record `gtdLocation` - (String,
  Optional) The GTD Location of the record on Global Traffic Director enabled
  domains; Unless GTD is enabled this should either be omitted or set to
  "DEFAULT"

Additional arguments are listed below under DNS Record Types.

## DNS Record Types

The type of record being created affects the interpretation of
the `value` argument; also, some additional arguments are
required for some record types.
http://help.dnsmadeeasy.com/tutorials/managed-dns/ has more
information.

#### A Record

* `value` is the hostname

#### CNAME Record

* `value` is the alias name

#### ANAME Record

* `value` is the aname target

#### MX Record

* `value` is the server
* `mxLevel` (Integer, Required) is the MX level

####  HTTPRED Record

* `value` is the URL
* `hardLink` (Boolean, Optional) If true, any request that is
  made for this record will have the path removed after the
  fully qualified domain name portion of the requested URL
* `redirectType` (Required) One of 'Hidden Frame Masked',
  'Standard 301', or 'Standard 302'
* `title` (Optional) If set, the hidden iframe that is
  used in conjunction with the Hidden Frame Masked Redirect
  Type will have the HTML meta description data field set to
  the value of this field
* `keywords` (Optional) If set, the hidden iframe that is used
  in conjunction with the Hidden Frame Masked Redirect Type
  will have the HTML meta keywords data field set to the value
  of this field
* `description` (Optional) A human-readable description.

#### TXT Record

* `value` is free form text

#### SPF Record

* `value` is the SPF definition of hosts allowed to send email

####  PTR Record

* `value` is the reverse DNS for the host

#### NS Record

* `value` is the host name of the server

#### AAAA Record

* `value` is the IPv6 address

#### SRV Record

* `value` is the host
* `priority` (Integer, Required). Acts the same way as MX Level
* `weight` (Integer, Required). Hits will be assigned proportionately
  by weight
* `port` (Integer, Required). The actual port of the service offered

## Attributes Reference

The following attributes are exported:

* `name` - The name of the record
* `type` - The type of the record
* `value` - The value of the record
  `type` (see below)
* `ttl` - The TTL of the record
* `gtdLocation` - The GTD Location of the record on GTD enabled domains

Additional fields may also be exported by some record types -
see DNS Record Types.

#### Record Type Examples

Following are examples of using each of the record types.

```hcl
# Provide your API and Secret Keys, and whether the sandbox
# is being used (defaults to false)
provider "dme" {
  akey       = "aaaaaa1a-11a1-1aa1-a101-11a1a11aa1aa"
  skey       = "11a0a11a-a1a1-111a-a11a-a11110a11111"
  usesandbox = true
}

# A Record
resource "dme_record" "testa" {
  domainid    = "123456"
  name        = "testa"
  type        = "A"
  value       = "1.1.1.1"
  ttl         = 1000
  gtdLocation = "DEFAULT"
}

# CNAME record
resource "dme_record" "testcname" {
  domainid = "123456"
  name     = "testcname"
  type     = "CNAME"
  value    = "foo"
  ttl      = 1000
}

# ANAME record
resource "dme_record" "testaname" {
  domainid = "123456"
  name     = "testaname"
  type     = "ANAME"
  value    = "foo"
  ttl      = 1000
}

# MX record
resource "dme_record" "testmx" {
  domainid = "123456"
  name     = "testmx"
  type     = "MX"
  value    = "foo"
  mxLevel  = 10
  ttl      = 1000
}

# HTTPRED
resource "dme_record" "testhttpred" {
  domainid     = "123456"
  name         = "testhttpred"
  type         = "HTTPRED"
  value        = "https://github.com/soniah/terraform-provider-dme"
  hardLink     = true
  redirectType = "Hidden Frame Masked"
  title        = "An Example"
  keywords     = "terraform example"
  description  = "This is a description"
  ttl          = 2000
}

# TXT record
resource "dme_record" "testtxt" {
  domainid = "123456"
  name     = "testtxt"
  type     = "TXT"
  value    = "foo"
  ttl      = 1000
}

# SPF record
resource "dme_record" "testspf" {
  domainid = "123456"
  name     = "testspf"
  type     = "SPF"
  value    = "foo"
  ttl      = 1000
}

# PTR record
resource "dme_record" "testptr" {
  domainid = "123456"
  name     = "testptr"
  type     = "PTR"
  value    = "foo"
  ttl      = 1000
}

# NS record
resource "dme_record" "testns" {
  domainid = "123456"
  name     = "testns"
  type     = "NS"
  value    = "foo"
  ttl      = 1000
}

# AAAA record
resource "dme_record" "testaaaa" {
  domainid = "123456"
  name     = "testaaaa"
  type     = "AAAA"
  value    = "FE80::0202:B3FF:FE1E:8329"
  ttl      = 1000
}

# SRV record
resource "dme_record" "testsrv" {
  domainid = "123456"
  name     = "testsrv"
  type     = "SRV"
  value    = "foo"
  priority = 10
  weight   = 20
  port     = 30
  ttl      = 1000
}
```
