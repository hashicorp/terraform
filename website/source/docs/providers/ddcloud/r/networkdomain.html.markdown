---
layout: "ddcloud"
page_title: "Dimension Data Managed Cloud Platform: networkdomain"
sidebar_current: "docs-ddcloud-resource-networkdomain"
description: |-
  Allows Terraform to manage a Managed Cloud Platform network domain.
---

# ddcloud\_networkdomain

A Network Domain is the fundamental building block for your MCP 2.0 Cloud deployment. At least one Network Domain and one VLAN must be deployed before you can deploy your first Server in an MCP 2.0 data center.

Refer to the documentation for further details:
https://community.opsourcecloud.net/View.jsp?procId=994fa801956149b3861e428801f9888f

## Example Usage

```
resource "ddcloud_networkdomain" "my-domain" {
    name                    = "terraform-test-domain"
    description             = "This is my Terraform test network domain."
    datacenter              = "AU9" # The ID of the data centre in which to create your network domain.
}
```

## Argument Reference

The following arguments are supported:

* `name` - (Required) A name for the network domain.
* `description` - (Optional) A description for the network domain.
* `plan` - (Optional) The plan (service level) for the network domain ("ESSENTIALS" or "ADVANCED", default is "ESSENTIALS").
* `datacenter` - (Required) The Id of the MCP 2.0 datacenter in which to create the network domain.

## Attributes Reference

The following attributes are exported:

* `nat_ipv4_address` - The IPv4 address for the network domain's IPv6->IPv4 Source Network Address Translation (SNAT). This is the IPv4 address of the network domain's IPv4 egress.
