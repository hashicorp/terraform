---
layout: "oneview"
page_title: "Oneview: icsp_server"
sidebar_current: "docs-oneview-icsp-server"
description: |-
  Hooks a server profile created in OneView into ICSP.
---

# oneview\_icsp\_server

Hooks a server profile created in OneView into ICSP.

## Example Usage

```js
resource "icsp_server" "default" {
  ilo_ip = "15.x.x.x"
  user_name = "ilo_user"
  password = "ilo_password"
  serial_number = "${oneview_server_profile.default.serial_number}"
  build_plans = ["/rest/os-deployment-build-plans/1570001"]
}
```

## Argument Reference

The following arguments are supported: 

* `ilo_ip` - (Required) The IP address of the iLO of the server.

* `user_name` - (Required) The user name required to log into the server's iLO.

* `password` - (Required) The password required to log into the server's iLO.

* `serial_number` - (Required) The serial number assigned to the Server.

- - -

* `port` - (Optional) The iLO port to use when logging in. 
  This defaults to 443.
  
* `build_plans` - (Optional) An array of build plan uris that you want to run on the server.

* `public_mac` - (Optional) The MAC address of the NIC that will be the public network connection.
  
* `public_slot_id` - (Optional) The slot id for the public network connection.

* `custom_attribute` - (Optional) A key/value pair for a custom attribute you would like associated with 
the server on icsp. Custom Attribute options specified below.

Custom Attribute supports the following:

* `key` - (Required) - The unique name of the attribute.

* `value` - (Required) - The value of the attribute.

* `scope` - (Optional) - The scope of the attribute. Defaults to `server`.

## Attributes Reference

In addition to the arguments listed above, the following computed attributes are exported:

* `mid` -  A unique ID assigned to the Server by Server Automation.

* `public_ipv4` - The public ip address if `public_mac` and `public_slot_id` were specified in the configuration.
