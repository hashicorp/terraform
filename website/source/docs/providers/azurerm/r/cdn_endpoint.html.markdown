---
layout: "azurerm"
page_title: "Azure Resource Manager: azurerm_cdn_endpoint"
sidebar_current: "docs-azurerm-resource-cdn-endpoint"
description: |-
  Create a CDN Endpoint entity.
---

# azurerm\_cdn\_endpoint

A CDN Endpoint is the entity within a CDN Profile containing configuration information regarding caching behaviors and origins. The CDN Endpoint is exposed using the URL format <endpointname>.azureedge.net by default, but custom domains can also be created.

## Example Usage

```hcl
resource "azurerm_resource_group" "test" {
  name     = "acceptanceTestResourceGroup1"
  location = "West US"
}

resource "azurerm_cdn_profile" "test" {
  name                = "acceptanceTestCdnProfile1"
  location            = "West US"
  resource_group_name = "${azurerm_resource_group.test.name}"
  sku                 = "Standard"
}

resource "azurerm_cdn_endpoint" "test" {
  name                = "acceptanceTestCdnEndpoint1"
  profile_name        = "${azurerm_cdn_profile.test.name}"
  location            = "West US"
  resource_group_name = "${azurerm_resource_group.test.name}"

  origin {
    name      = "acceptanceTestCdnOrigin1"
    host_name = "www.example.com"
  }
}
```

## Argument Reference

The following arguments are supported:

* `name` - (Required) Specifies the name of the CDN Endpoint. Changing this forces a
    new resource to be created.

* `resource_group_name` - (Required) The name of the resource group in which to
    create the CDN Endpoint.

* `profile_name` - (Required) The CDN Profile to which to attach the CDN Endpoint.

* `location` - (Required) Specifies the supported Azure location where the resource exists. Changing this forces a new resource to be created.

* `origin_host_header` - (Optional) The host header CDN provider will send along with content requests to origins. Defaults to the host name of the origin.

* `is_http_allowed` - (Optional) Defaults to `true`.

* `is_https_allowed` - (Optional) Defaults to `true`.

* `origin` - (Optional) The set of origins of the CDN endpoint. When multiple origins exist, the first origin will be used as primary and rest will be used as failover options.
Each `origin` block supports fields documented below.

* `origin_path` - (Optional) The path used at for origin requests.

* `querystring_caching_behaviour` - (Optional) Sets query string caching behavior. Allowed values are `IgnoreQueryString`, `BypassCaching` and `UseQueryString`. Defaults to `IgnoreQueryString`.

* `content_types_to_compress` - (Optional) An array of strings that indicates a content types on which compression will be applied. The value for the elements should be MIME types.

* `is_compression_enabled` - (Optional) Indicates whether compression is to be enabled. Defaults to false.

* `tags` - (Optional) A mapping of tags to assign to the resource.

The `origin` block supports:

* `name` - (Required) The name of the origin. This is an arbitrary value. However, this value needs to be unique under endpoint.

* `host_name` - (Required) A string that determines the hostname/IP address of the origin server. This string could be a domain name, IPv4 address or IPv6 address.

* `http_port` - (Optional) The HTTP port of the origin. Defaults to null. When null, 80 will be used for HTTP.

* `https_port` - (Optional) The HTTPS port of the origin. Defaults to null. When null, 443 will be used for HTTPS.

## Attributes Reference

The following attributes are exported:

* `id` - The CDN Endpoint ID.

## Import

CDN Endpoints can be imported using the `resource id`, e.g.

```
terraform import azurerm_cdn_endpoint.test /subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/mygroup1/providers/Microsoft.Cdn/profiles/myprofile1/endpoints/myendpoint1
```