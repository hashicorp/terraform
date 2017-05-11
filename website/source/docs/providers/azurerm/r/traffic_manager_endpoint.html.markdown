---
layout: "azurerm"
page_title: "Azure Resource Manager: azurerm_traffic_manager_endpoint"
sidebar_current: "docs-azurerm-resource-network-traffic-manager-endpoint"
description: |-
  Creates a Traffic Manager Endpoint.
---

# azurerm\_traffic\_manager\_endpoint

Creates a Traffic Manager Endpoint.

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

resource "azurerm_traffic_manager_endpoint" "test" {
  name                = "profile1"
  resource_group_name = "${azurerm_resource_group.test.name}"
  profile_name        = "${azurerm_traffic_manager_profile.test.name}"
  target              = "terraform.io"
  type                = "externalEndpoints"
  weight              = 100
}
```

## Argument Reference

The following arguments are supported:

* `name` - (Required) The name of the Traffic Manager endpoint. Changing this forces a
    new resource to be created.

* `resource_group_name` - (Required) The name of the resource group in which to
    create the Traffic Manager endpoint.

* `profile_name` - (Required) The name of the Traffic Manager Profile to attach
    create the Traffic Manager endpoint.

* `endpoint_status` - (Optional) The status of the Endpoint, can be set to
    either `Enabled` or `Disabled`. Defaults to `Enabled`.

* `type` - (Required) The Endpoint type, must be one of:
    - `azureEndpoints`
    - `externalEndpoints`
    - `nestedEndpoints`

* `target` - (Optional) The FQDN DNS name of the target. This argument must be
    provided for an endpoint of type `externalEndpoints`, for other types it
    will be computed.

* `target_resource_id` - (Optional) The resource id of an Azure resource to
    target. This argument must be provided for an endpoint of type
    `azureEndpoints` or `nestedEndpoints`.

* `weight` - (Optional) Specifies how much traffic should be distributed to this
    endpoint, this must be specified for Profiles using the  `Weighted` traffic
    routing method. Supports values between 1 and 1000.

* `priority` - (Optional) Specifies the priority of this Endpoint, this must be
    specified for Profiles using the `Priority` traffic routing method. Supports
    values between 1 and 1000, with no Endpoints sharing the same value. If
    omitted the value will be computed in order of creation.

* `endpoint_location` - (Optional) Specifies the Azure location of the Endpoint,
    this must be specified for Profiles using the `Performance` routing method
    if the Endpoint is of either type `nestedEndpoints` or `externalEndpoints`.
    For Endpoints of type `azureEndpoints` the value will be taken from the
    location of the Azure target resource.

* `min_child_endpoints` - (Optional) This argument specifies the minimum number
    of endpoints that must be ‘online’ in the child profile in order for the
    parent profile to direct traffic to any of the endpoints in that child
    profile. This argument only applies to Endpoints of type `nestedEndpoints`
    and defaults to `1`.

## Attributes Reference

The following attributes are exported:

* `id` - The Traffic Manager Endpoint id.

## Import

Traffic Manager Endpoints can be imported using the `resource id`, e.g.

```
terraform import azurerm_traffic_manager_endpoint.testEndpoints /subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/mygroup1/providers/Microsoft.Network/trafficManagerProfiles/mytrafficmanagerprofile1/azureEndpoints/mytrafficmanagerendpoint
```