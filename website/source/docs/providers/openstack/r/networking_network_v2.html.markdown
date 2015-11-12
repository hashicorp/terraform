---
layout: "openstack"
page_title: "OpenStack: openstack_networking_network_v2"
sidebar_current: "docs-openstack-resource-networking-network-v2"
description: |-
  Manages a V2 Neutron network resource within OpenStack.
---

# openstack\_networking\_network_v2

Manages a V2 Neutron network resource within OpenStack.

## Example Usage

```
resource "openstack_networking_network_v2" "network_1" {
  name = "network_1"
  admin_state_up = "true"
}

resource "openstack_networking_subnet_v2" "subnet_1" {
  name = "subnet_1"
  network_id = "${openstack_networking_network_v2.network_1.id}"
  cidr = "192.168.199.0/24"
  ip_version = 4
}

resource "openstack_compute_secgroup_v2" "secgroup_1" {
  name = "secgroup_1"
  description = "a security group"
  rule {
    from_port = 22
    to_port = 22
    ip_protocol = "tcp"
    cidr = "0.0.0.0/0"
  }
}

resource "openstack_networking_port_v2" "port_1" {
  name = "port_1"
  network_id = "${openstack_networking_network_v2.network_1.id}"
  admin_state_up = "true"
  security_groups = ["${openstack_compute_secgroup_v2.secgroup_1.id}"]

  fixed_ips {
      "subnet_id" =  "008ba151-0b8c-4a67-98b5-0d2b87666062"
      "ip_address" =  "172.24.4.2"
  }
}

resource "openstack_compute_instance_v2" "instance_1" {
  name = "instance_1"
  security_groups = ["${openstack_compute_secgroup_v2.secgroup_1.name}"]

  network {
    port = "${openstack_networking_port_v2.port_1.id}"
  }
}

```

## Argument Reference

The following arguments are supported:

* `region` - (Required) The region in which to obtain the V2 Networking client.
    A Networking client is needed to create a Neutron network. If omitted, the
    `OS_REGION_NAME` environment variable is used. Changing this creates a new
    network.

* `name` - (Optional) The name of the network. Changing this updates the name of
    the existing network.

* `shared` - (Optional)  Specifies whether the network resource can be accessed
    by any tenant or not. Changing this updates the sharing capabalities of the
    existing network.

* `tenant_id` - (Optional) The owner of the network. Required if admin wants to
    create a network for another tenant. Changing this creates a new network.

* `admin_state_up` - (Optional) The administrative state of the network.
    Acceptable values are "true" and "false". Changing this value updates the
    state of the existing network.

## Attributes Reference

The following attributes are exported:

* `region` - See Argument Reference above.
* `name` - See Argument Reference above.
* `shared` - See Argument Reference above.
* `tenant_id` - See Argument Reference above.
* `admin_state_up` - See Argument Reference above.
