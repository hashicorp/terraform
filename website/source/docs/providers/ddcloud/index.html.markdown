---
layout: "ddcloud"
page_title: "Provider: Dimension Data Managed Cloud Platform"
sidebar_current: "docs-ddcloud-index"
description: |-
  The Dimension Data Managed Cloud Platform provider is used to interact with Dimension Data's Managed Cloud Platform resources.
---

# Managed Cloud Platform

Managed Cloud Platform is Dimension Data's cloud virtualisation service. The API for controlling the Managed Cloud Platform is called CloudControl.
Use the navigation to the left to read about the available resources.

## Example Usage

```
/*
 * This configuration will create a single server running CentOS and expose it to the internet on port 80.
 *
 * By default, the Managed Cloud Platform's CentOS image does not have httpd installed (`yum install httpd`) so there should be no problem exposing port 80.
 */

provider "ddcloud" {
	# You don't have to specify username or password if the DD_COMPUTE_USER and DD_COMPUTE_PASSWORD environment variables are set.
	"username"              = "my_username"
    "password"              = "my_password" # Watch out for escaping if your password contains special characters such as "$".
    "region"                = "AU" # The DD compute region code (e.g. "AU", "NA", "EU")
}

# The network domain that contains all the resources in this example.
resource "ddcloud_networkdomain" "my-domain" {
    name                    = "terraform-test-domain"
    description             = "This is my Terraform test network domain."
    datacenter              = "AU9" # The ID of the data centre in which to create your network domain.
}

# The VLAN that my-server is connected to.
resource "ddcloud_vlan" "my-vlan" {
    name                    = "terraform-test-vlan"
    description             = "This is my Terraform test VLAN."

    networkdomain           = "${ddcloud_networkdomain.my-domain.id}"

    # VLAN's default network: 192.168.17.1 -> 192.168.17.254 (netmask = 255.255.255.0)
    ipv4_base_address       = "192.168.17.0"
    ipv4_prefix_size        = 24

    depends_on              = [ "ddcloud_networkdomain.my-domain"]
}

# A Server (virtual machine) running CentOS 7.
resource "ddcloud_server" "my-server" {
    name                    = "terraform-server"
    description             = "This is my Terraform test server."
    admin_password          = "password"

    memory_gb               = 8

    networkdomain           = "${ddcloud_networkdomain.test-domain.id}"
    primary_adapter_ipv4    = "192.168.17.10"
    dns_primary             = "8.8.8.8"
    dns_secondary           = "8.8.4.4"

    osimage_name            = "CentOS 7 64-bit 2 CPU"

    depends_on              = [ "ddcloud_vlan.my-vlan" ]
}

# A NAT rule forwarding traffic from a public IPv4 address to my-server.
resource "ddcloud_nat" "my-server-nat" {
    networkdomain           = "${ddcloud_networkdomain.my-domain.id}"
    private_ipv4            = "${ddcloud_server.my-server.primary_adapter_ipv4}"

    # public_ipv4 is computed at deploy time.

    depends_on              = [ "ddcloud_vlan.test-vlan" ]
}

# A firewall rule permitting HTTP traffic to my-server's public IPv4 address.
resource "ddcloud_firewall_rule" "test-vm-http-in" {
    name                    = "my_server.HTTP.Inbound"
    placement               = "first"
    action                  = "accept" # Valid values are "accept" or "drop."
    enabled                 = true

    ip_version              = "ipv4"
    protocol                = "tcp"

    # source_address is computed at deploy time (not specified = "any").
    # source_port is computed at deploy time (not specified = "any).
    # You can also specify source_network (e.g. 10.2.198.0/24) or source_address_list instead of source_address.
    # For a ddcloud_vlan, you can obtain these values using the ipv4_baseaddress and ipv4_prefixsize properties.

    # You can also specify destination_network or destination_address_list instead of source_address.
    destination_address     = "${ddcloud_nat.my-server-nat.public_ipv4}"
    destination_port        = "80"

    networkdomain           = "${ddcloud_networkdomain.my-domain.id}"
}
```

## Argument Reference

The following arguments are supported:

* `username` - (Optional) The user name for authenticating to CloudControl.
* `password` - (Optional) The password for authenticating to CloudControl.
* `region` - (Optional) The Managed Cloud Platform region code (e.g. 'AU' - Australia, 'EU' - Europe, 'NA' - North America) that identifies the CloudControl end-point to connect to.
