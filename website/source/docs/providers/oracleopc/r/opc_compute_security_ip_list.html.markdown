---
layout: "oracle"
page_title: "Oracle: opc_compute_security_ip_list"
sidebar_current: "docs-opc-resource-security-list"
description: |-
  Creates and manages a security IP list in an OPC identity domain.
---

# opc\_compute\_ip\_reservation

The ``opc_compute_security_ip_list`` resource creates and manages a security IP list in an OPC identity domain.

## Example Usage

```
resource "opc_compute_security_ip_list" "sec_ip_list1" {
	name = "sec-ip-list1"
	ip_entries = ["217.138.34.4"]
}
```

## Argument Reference

The following arguments are supported:

* `name` - (Required) The unique (within the identity domain) name of the security IP list.

* `ip_entries` - (Required) The IP addresses to include in the list.