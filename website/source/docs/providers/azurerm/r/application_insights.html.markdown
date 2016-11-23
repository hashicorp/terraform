---
layout: "azurerm"
page_title: "Azure Resource Manager: azurerm_application_insights"
sidebar_current: "docs-azurerm-resource-application-insights"
description: |-
  Create an Application Insights instance.
---

# azurerm\_application\_insights

Create an Application Insights instance.

## Example Usage

```
resource "azurerm_resource_group" "test" {
    name = "resourceGroup1"
    location = "West US"
}

resource "azurerm_application_insights" "test" {
    name                = "myinsights"
    location            = "${azurerm_resource_group.test.location}"
    resource_group_name = "${azurerm_resource_group.test.name}"
    application_id      = "myapplicationid"
    application_type    = "web"
    tags {
        environment = "Production"
    }
}
```

## Argument Reference

The following arguments are supported:

* `name` - (Required) Specifies the name of the Application Insights instance. Changing this forces a
    new resource to be created.

* `resource_group_name` - (Required) The name of the resource group in which to
    create the Application Insights instance.

* `location` - (Required) Specifies the supported Azure location where the resource exists. Changing this forces a new resource to be created.

* `application_id` - (Required) Specifies the Application ID you want to assign to the Application Insights instance.

* `application_type` - (Required) Specifies the Application Type you wish to use - either `web` or `other`.

* `tags` - (Optional) A mapping of tags to assign to the resource.

## Attributes Reference

The following attributes are exported:

* `id` - The Application Insights instance ID.

* `instrumentation_key` - The Application Insights Instrumentation Key.

* `app_id` - The Application Insights App ID.


## Import

Availability Sets can be imported using the `resource id`, e.g.

```
terraform import azurerm_application_insights.insights1 /subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/mygroup1/providers/Microsoft.Insights/components/insights1
```
