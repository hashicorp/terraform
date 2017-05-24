---
layout: "librato"
page_title: "Librato: librato_service"
sidebar_current: "docs-librato-resource-service"
description: |-
  Provides a Librato service resource. This can be used to create and manage notification services on Librato.
---

# librato\_service

Provides a Librato Service resource. This can be used to
create and manage notification services on Librato.

## Example Usage

```hcl
# Create a new Librato service
resource "librato_service" "email" {
  title = "Email the admins"
  type  = "mail"

  settings = <<EOF
{
  "addresses": "admin@example.com"
}
EOF
}
```

## Argument Reference

The following arguments are supported. Please check the [relevant documentation](https://github.com/librato/librato-services/tree/master/services) for each type of alert.

* `type` - (Required) The type of notificaion.
* `title` - (Required) The alert title.
* `settings` - (Required) a JSON hash of settings specific to the alert type.

## Attributes Reference

The following attributes are exported:

* `id` - The ID of the alert.
* `type` - The type of notificaion.
* `title` - The alert title.
* `settings` - a JSON hash of settings specific to the alert type.
