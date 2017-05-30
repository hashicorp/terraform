---
layout: "nsx"
page_title: "VMware NSX: nsx_service"
sidebar_current: "docs-nsx-resource-nsx-service"
description: |-
  Provides a VMware NSX service resource. This can be used to create, modify, and delete NSX service resources.
---

# nsx\_service

Provides a VMware NSX service resource. This can be used to create,
modify, and delete NSX services.

## Example Usage

```hcl
resource "nsx_service" "foo" {
    name = "foo_service_http_80"
    scopeid = "globalroot-0"
    desc = "FOO TCP port 80 - http"
    proto = "TCP"
    ports = "80"
}
```

## Argument Reference

The following arguments are supported:
* `name` - (Required) The name you want to call this service by.
* `scopeid` - (Required) The scopeid.
* `description` - (Required) Description of the service.
* `protocol` - (Required) The chosen protocol. e.g. TCP, ICMP.
* `ports` - (Required) The ports assigned to this service. i.e 80,8080
