---
layout: "azurerm"
page_title: "Azure Resource Manager: azure_virtual_machine_extension"
sidebar_current: "docs-azurerm-resource-virtualmachine-extension"
description: |-
    Creates a new Virtual Machine Extension to provide post deployment
    configuration and run automated tasks.
---

# azurerm\_virtual\_machine\_extension

Creates a new Virtual Machine Extension to provide post deployment configuration
and run automated tasks.

## Example Usage

```hcl
resource "azurerm_resource_group" "test" {
  name     = "acctestrg"
  location = "West US"
}

resource "azurerm_virtual_network" "test" {
  name                = "acctvn"
  address_space       = ["10.0.0.0/16"]
  location            = "West US"
  resource_group_name = "${azurerm_resource_group.test.name}"
}

resource "azurerm_subnet" "test" {
  name                 = "acctsub"
  resource_group_name  = "${azurerm_resource_group.test.name}"
  virtual_network_name = "${azurerm_virtual_network.test.name}"
  address_prefix       = "10.0.2.0/24"
}

resource "azurerm_network_interface" "test" {
  name                = "acctni"
  location            = "West US"
  resource_group_name = "${azurerm_resource_group.test.name}"

  ip_configuration {
    name                          = "testconfiguration1"
    subnet_id                     = "${azurerm_subnet.test.id}"
    private_ip_address_allocation = "dynamic"
  }
}

resource "azurerm_storage_account" "test" {
  name                = "accsa"
  resource_group_name = "${azurerm_resource_group.test.name}"
  location            = "westus"
  account_type        = "Standard_LRS"

  tags {
    environment = "staging"
  }
}

resource "azurerm_storage_container" "test" {
  name                  = "vhds"
  resource_group_name   = "${azurerm_resource_group.test.name}"
  storage_account_name  = "${azurerm_storage_account.test.name}"
  container_access_type = "private"
}

resource "azurerm_virtual_machine" "test" {
  name                  = "acctvm"
  location              = "West US"
  resource_group_name   = "${azurerm_resource_group.test.name}"
  network_interface_ids = ["${azurerm_network_interface.test.id}"]
  vm_size               = "Standard_A0"

  storage_image_reference {
    publisher = "Canonical"
    offer     = "UbuntuServer"
    sku       = "14.04.2-LTS"
    version   = "latest"
  }

  storage_os_disk {
    name          = "myosdisk1"
    vhd_uri       = "${azurerm_storage_account.test.primary_blob_endpoint}${azurerm_storage_container.test.name}/myosdisk1.vhd"
    caching       = "ReadWrite"
    create_option = "FromImage"
  }

  os_profile {
    computer_name  = "hostname"
    admin_username = "testadmin"
    admin_password = "Password1234!"
  }

  os_profile_linux_config {
    disable_password_authentication = false
  }

  tags {
    environment = "staging"
  }
}

resource "azurerm_virtual_machine_extension" "test" {
  name                 = "hostname"
  location             = "West US"
  resource_group_name  = "${azurerm_resource_group.test.name}"
  virtual_machine_name = "${azurerm_virtual_machine.test.name}"
  publisher            = "Microsoft.OSTCExtensions"
  type                 = "CustomScriptForLinux"
  type_handler_version = "1.2"

  settings = <<SETTINGS
	{
		"commandToExecute": "hostname"
	}
SETTINGS

  tags {
    environment = "Production"
  }
}
```

## Argument Reference

The following arguments are supported:

* `name` - (Required) The name of the virtual machine extension peering. Changing
    this forces a new resource to be created.

* `location` - (Required) The location where the extension is created. Changing
    this forces a new resource to be created.

* `resource_group_name` - (Required) The name of the resource group in which to
    create the virtual network. Changing this forces a new resource to be
    created.

* `virtual_machine_name` - (Required) The name of the virtual machine. Changing
    this forces a new resource to be created.

* `publisher` - (Required) The publisher of the extension, available publishers
    can be found by using the Azure CLI.

* `type` - (Required) The type of extension, available types for a publisher can
    be found using the Azure CLI.

* `type_handler_version` - (Required) Specifies the version of the extension to
    use, available versions can be found using the Azure CLI.

* `auto_upgrade_minor_version` - (Optional) Specifies if the platform deploys
    the latest minor version update to the `type_handler_version` specified.

* `settings` - (Required) The settings passed to the extension, these are
    specified as a JSON object in a string.

* `protected_settings` - (Optional) The protected_settings passed to the
    extension, like settings, these are specified as a JSON object in a string.

## Attributes Reference

The following attributes are exported:

* `id` - The Virtual Machine Extension ID.

## Import

Virtual Machine Extensions can be imported using the `resource id`, e.g.

```
terraform import azurerm_virtual_machine_extension.test /subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/mygroup1/providers/Microsoft.Compute/virtualMachines/myVM/extensions/hostname
```