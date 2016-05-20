---
layout: "openstack"
page_title: "OpenStack: openstack_networking_subnet_v2"
sidebar_current: "docs-openstack-resource-networking-subnet-v2"
description: |-
  Manages a V2 Neutron subnet resource within OpenStack.
---

# openstack\_networking\_subnet_v2

Manages a V2 Neutron subnet resource within OpenStack.

## Example Usage

```
resource "openstack_networking_network_v2" "network_1" {
  name = "tf_test_network"
  admin_state_up = "true"
}

resource "openstack_networking_subnet_v2" "subnet_1" {
  network_id = "${openstack_networking_network_v2.network_1.id}"
  cidr = "192.168.199.0/24"
}
```

## Argument Reference

The following arguments are supported:

* `region` - (Required) The region in which to obtain the V2 Networking client.
    A Networking client is needed to create a Neutron subnet. If omitted, the
    `OS_REGION_NAME` environment variable is used. Changing this creates a new
    subnet.

* `network_id` - (Required) The UUID of the parent network. Changing this
    creates a new subnet.

* `cidr` - (Required) CIDR representing IP range for this subnet, based on IP
    version. Changing this creates a new subnet.

* `ip_version` - (Optional) IP version, either 4 (default) or 6. Changing this creates a
    new subnet.

* `name` - (Optional) The name of the subnet. Changing this updates the name of
    the existing subnet.

* `tenant_id` - (Optional) The owner of the subnet. Required if admin wants to
    create a subnet for another tenant. Changing this creates a new subnet.

* `allocation_pools` - (Optional) An array of sub-ranges of CIDR available for
    dynamic allocation to ports. The allocation_pool object structure is
    documented below. Changing this creates a new subnet.

* `gateway_ip` - (Optional)  Default gateway used by devices in this subnet.
    Leaving this blank and not setting `no_gateway` will cause a default
    gateway of `.1` to be used. Changing this updates the gateway IP of the
    existing subnet.

* `no_gateway` - (Optional) Do not set a gateway IP on this subnet. Changing
    this removes or adds a default gateway IP of the existing subnet.

* `enable_dhcp` - (Optional) The administrative state of the network.
    Acceptable values are "true" and "false". Changing this value enables or
    disables the DHCP capabilities of the existing subnet.

* `dns_nameservers` - (Optional) An array of DNS name server names used by hosts
    in this subnet. Changing this updates the DNS name servers for the existing
    subnet.

* `host_routes` - (Optional) An array of routes that should be used by devices
    with IPs from this subnet (not including local subnet route). The host_route
    object structure is documented below. Changing this updates the host routes
    for the existing subnet.

The `allocation_pools` block supports:

* `start` - (Required) The starting address.

* `end` - (Required) The ending address.

The `host_routes` block supports:

* `destination_cidr` - (Required) The destination CIDR.

* `next_hop` - (Required) The next hop in the route.

## Attributes Reference

The following attributes are exported:

* `region` - See Argument Reference above.
* `network_id` - See Argument Reference above.
* `cidr` - See Argument Reference above.
* `ip_version` - See Argument Reference above.
* `name` - See Argument Reference above.
* `tenant_id` - See Argument Reference above.
* `allocation_pools` - See Argument Reference above.
* `gateway_ip` - See Argument Reference above.
* `enable_dhcp` - See Argument Reference above.
* `dns_nameservers` - See Argument Reference above.
* `host_routes` - See Argument Reference above.
