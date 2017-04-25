---
layout: "azurerm"
page_title: "Azure Resource Manager: azurerm_traffic_manager_profile"
sidebar_current: "docs-azurerm-resource-network-traffic-manager-profile"
description: |-
  Creates a Traffic Manager Profile.
---

# azurerm\_traffic\_manager\_profile

Creates a Traffic Manager Profile to which multiple endpoints can be attached.

## Example Usage

```hcl
resource "azurerm_traffic_manager_profile" "test" {
  name                = "profile1"
  resource_group_name = "${azurerm_resource_group.test.name}"
  location            = "West US"

  traffic_routing_method = "Weighted"

  dns_config {
    relative_name = "profile1"
    ttl           = 100
  }

  monitor_config {
    protocol = "http"
    port     = 80
    path     = "/"
  }

  tags {
    environment = "Production"
  }
}
```

## Argument Reference

The following arguments are supported:

* `name` - (Required) The name of the virtual network. Changing this forces a
    new resource to be created.

* `resource_group_name` - (Required) The name of the resource group in which to
    create the virtual network.

* `profile_status` - (Optional) The status of the profile, can be set to either
    `Enabled` or `Disabled`. Defaults to `Enabled`.

* `traffic_routing_method` - (Required) Specifies the algorithm used to route
    traffic, possible values are:
    - `Performance`- Traffic is routed via the User's closest Endpoint
    - `Weighted` - Traffic is spread across Endpoints proportional to their
        `weight` value.
    - `Priority` - Traffic is routed to the Endpoint with the lowest
        `priority` value.

* `dns_config` - (Required) This block specifies the DNS configuration of the
    Profile, it supports the fields documented below.

* `monitor_config` - (Required) This block specifies the Endpoint monitoring
    configuration for the Profile, it supports the fields documented below.

* `tags` - (Optional) A mapping of tags to assign to the resource.

The `dns_config` block supports:

* `relative_name` - (Required) The relative domain name, this is combined with
    the domain name used by Traffic Manager to form the FQDN which is exported
    as documented below. Changing this forces a new resource to be created.

* `ttl` - (Required) The TTL value of the Profile used by Local DNS resolvers
    and clients.

The `monitor_config` block supports:

* `http` - (Required) The protocol used by the monitoring checks, supported
    values are `http` or `https`.

* `port` - (Required) The port number used by the monitoring checks.

* `path` - (Required) The path used by the monitoring checks.

## Attributes Reference

The following attributes are exported:

* `id` - The Traffic Manager Profile id.
* `fqdn` - The FQDN of the created Profile.

## Notes

The Traffic Manager is created with the location `global`.

## Import

Traffic Manager Profiles can be imported using the `resource id`, e.g.

```
terraform import azurerm_traffic_manager_profile.testProfile /subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/mygroup1/providers/Microsoft.Network/trafficManagerProfiles/mytrafficmanagerprofile1
```