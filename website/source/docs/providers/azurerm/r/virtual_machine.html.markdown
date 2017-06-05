---
layout: "azurerm"
page_title: "Azure Resource Manager: azurerm_virtual_machine"
sidebar_current: "docs-azurerm-resource-virtual-machine"
description: |-
  Create a Virtual Machine.
---

# azurerm\_virtual\_machine

Create a virtual machine.

## Example Usage (Unmanaged Disks)

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
```

## Example Usage With Additional Empty Data Disk (Unmanaged Disks)

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

  storage_data_disk {
    name          = "datadisk0"
    vhd_uri       = "${azurerm_storage_account.test.primary_blob_endpoint}${azurerm_storage_container.test.name}/datadisk0.vhd"
    disk_size_gb  = "1023"
    create_option = "Empty"
    lun           = 0
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
```

## Example Usage (Managed Disks)

```hcl
resource "azurerm_resource_group" "test" {
  name     = "acctestrg"
  location = "West US 2"
}

resource "azurerm_virtual_network" "test" {
  name                = "acctvn"
  address_space       = ["10.0.0.0/16"]
  location            = "West US 2"
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
  location            = "West US 2"
  resource_group_name = "${azurerm_resource_group.test.name}"

  ip_configuration {
    name                          = "testconfiguration1"
    subnet_id                     = "${azurerm_subnet.test.id}"
    private_ip_address_allocation = "dynamic"
  }
}

resource "azurerm_managed_disk" "test" {
  name                 = "datadisk_existing"
  location             = "West US 2"
  resource_group_name  = "${azurerm_resource_group.test.name}"
  storage_account_type = "Standard_LRS"
  create_option        = "Empty"
  disk_size_gb         = "1023"
}

resource "azurerm_virtual_machine" "test" {
  name                  = "acctvm"
  location              = "West US 2"
  resource_group_name   = "${azurerm_resource_group.test.name}"
  network_interface_ids = ["${azurerm_network_interface.test.id}"]
  vm_size               = "Standard_DS1_v2"

  storage_image_reference {
    publisher = "Canonical"
    offer     = "UbuntuServer"
    sku       = "14.04.2-LTS"
    version   = "latest"
  }

  storage_os_disk {
    name              = "myosdisk1"
    caching           = "ReadWrite"
    create_option     = "FromImage"
    managed_disk_type = "Standard_LRS"
  }

  storage_data_disk {
    name              = "datadisk_new"
    managed_disk_type = "Standard_LRS"
    create_option     = "Empty"
    lun               = 0
    disk_size_gb      = "1023"
  }

  storage_data_disk {
    name            = "${azurerm_managed_disk.test.name}"
    managed_disk_id = "${azurerm_managed_disk.test.id}"
    create_option   = "Attach"
    lun             = 1
    disk_size_gb    = "${azurerm_managed_disk.test.disk_size_gb}"
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
```

## Argument Reference

The following arguments are supported:

* `name` - (Required) Specifies the name of the virtual machine resource. Changing this forces a
    new resource to be created.
* `resource_group_name` - (Required) The name of the resource group in which to
    create the virtual machine.
* `location` - (Required) Specifies the supported Azure location where the resource exists. Changing this forces a new resource to be created.
* `plan` - (Optional) A plan block as documented below.
* `availability_set_id` - (Optional) The Id of the Availability Set in which to create the virtual machine
* `boot_diagnostics` - (Optional) A boot diagnostics profile block as referenced below.
* `vm_size` - (Required) Specifies the [size of the virtual machine](https://azure.microsoft.com/en-us/documentation/articles/virtual-machines-size-specs/).
* `storage_image_reference` - (Optional) A Storage Image Reference block as documented below.
* `storage_os_disk` - (Required) A Storage OS Disk block as referenced below.
* `delete_os_disk_on_termination` - (Optional) Flag to enable deletion of the OS disk VHD blob or managed disk when the VM is deleted, defaults to `false`
* `storage_data_disk` - (Optional) A list of Storage Data disk blocks as referenced below.
* `delete_data_disks_on_termination` - (Optional) Flag to enable deletion of storage data disk VHD blobs or managed disks when the VM is deleted, defaults to `false`
* `os_profile` - (Optional) An OS Profile block as documented below. Required when `create_option` in the `storage_os_disk` block is set to `FromImage`.

* `license_type` - (Optional, when a windows machine) Specifies the Windows OS license type. The only allowable value, if supplied, is `Windows_Server`.
* `os_profile_windows_config` - (Required, when a windows machine) A Windows config block as documented below.
* `os_profile_linux_config` - (Required, when a linux machine) A Linux config block as documented below.
* `os_profile_secrets` - (Optional) A collection of Secret blocks as documented below.
* `network_interface_ids` - (Required) Specifies the list of resource IDs for the network interfaces associated with the virtual machine.
* `primary_network_interface_id` - (Optional) Specifies the resource ID for the primary network interface associated with the virtual machine.
* `tags` - (Optional) A mapping of tags to assign to the resource.

For more information on the different example configurations, please check out the [azure documentation](https://msdn.microsoft.com/en-us/library/mt163591.aspx#Anchor_2)

`Plan` supports the following:

* `name` - (Required) Specifies the name of the image from the marketplace.
* `publisher` - (Optional) Specifies the publisher of the image.
* `product` - (Optional) Specifies the product of the image from the marketplace.

`boot_diagnostics` supports the following:

* `enabled`: (Required) Whether to enable boot diagnostics for the virtual machine.
* `storage_uri`: (Required) Blob endpoint for the storage account to hold the virtual machine's diagnostic files. This must be the root of a storage account, and not a storage container.

`storage_image_reference` supports the following:

* `publisher` - (Required) Specifies the publisher of the image used to create the virtual machine. Changing this forces a new resource to be created.
* `offer` - (Required) Specifies the offer of the image used to create the virtual machine. Changing this forces a new resource to be created.
* `sku` - (Required) Specifies the SKU of the image used to create the virtual machine. Changing this forces a new resource to be created.
* `version` - (Optional) Specifies the version of the image used to create the virtual machine. Changing this forces a new resource to be created.

`storage_os_disk` supports the following:

* `name` - (Required) Specifies the disk name.
* `vhd_uri` - (Optional) Specifies the vhd uri. Changing this forces a new resource to be created. Cannot be used with managed disks.
* `managed_disk_type` - (Optional) Specifies the type of managed disk to create. Value you must be either `Standard_LRS` or `Premium_LRS`. Cannot be used when `vhd_uri` is specified.
* `managed_disk_id` - (Optional) Specifies an existing managed disk to use by id. Can only be used when `create_option` is `Attach`. Cannot be used when `vhd_uri` is specified.
* `create_option` - (Required) Specifies how the virtual machine should be created. Possible values are `Attach` (managed disks only) and `FromImage`.
* `caching` - (Optional) Specifies the caching requirements.
* `image_uri` - (Optional) Specifies the image_uri in the form publisherName:offer:skus:version. `image_uri` can also specify the [VHD uri](https://azure.microsoft.com/en-us/documentation/articles/virtual-machines-linux-cli-deploy-templates/#create-a-custom-vm-image) of a custom VM image to clone. When cloning a custom disk image the `os_type` documented below becomes required.
* `os_type` - (Optional) Specifies the operating system Type, valid values are windows, linux.
* `disk_size_gb` - (Optional) Specifies the size of the os disk in gigabytes.

`storage_data_disk` supports the following:

* `name` - (Required) Specifies the name of the data disk.
* `vhd_uri` - (Optional) Specifies the uri of the location in storage where the vhd for the virtual machine should be placed. Cannot be used with managed disks.
* `managed_disk_type` - (Optional) Specifies the type of managed disk to create. Value you must be either `Standard_LRS` or `Premium_LRS`. Cannot be used when `vhd_uri` is specified.
* `managed_disk_id` - (Optional) Specifies an existing managed disk to use by id. Can only be used when `create_option` is `Attach`. Cannot be used when `vhd_uri` is specified.
* `create_option` - (Required) Specifies how the data disk should be created. Possible values are `Attach`, `FromImage` and `Empty`.
* `disk_size_gb` - (Required) Specifies the size of the data disk in gigabytes.
* `caching` - (Optional) Specifies the caching requirements.
* `lun` - (Required) Specifies the logical unit number of the data disk.

`os_profile` supports the following:

* `computer_name` - (Required) Specifies the name of the virtual machine.
* `admin_username` - (Required) Specifies the name of the administrator account.
* `admin_password` - (Required) Specifies the password of the administrator account.
* `custom_data` - (Optional) Specifies custom data to supply to the machine. On linux-based systems, this can be used as a cloud-init script. On other systems, this will be copied as a file on disk. Internally, Terraform will base64 encode this value before sending it to the API. The maximum length of the binary array is 65535 bytes.

~> **NOTE:** `admin_password` must be between 6-72 characters long and must satisfy at least 3 of password complexity requirements from the following:
1. Contains an uppercase character
2. Contains a lowercase character
3. Contains a numeric digit
4. Contains a special character

`os_profile_windows_config` supports the following:

* `provision_vm_agent` - (Optional)
* `enable_automatic_upgrades` - (Optional)
* `winrm` - (Optional) A collection of WinRM configuration blocks as documented below.
* `additional_unattend_config` - (Optional) An Additional Unattended Config block as documented below.

`winrm` supports the following:

* `protocol` - (Required) Specifies the protocol of listener
* `certificate_url` - (Optional) Specifies URL of the certificate with which new Virtual Machines is provisioned.

`additional_unattend_config` supports the following:

* `pass` - (Required) Specifies the name of the pass that the content applies to. The only allowable value is `oobeSystem`.
* `component` - (Required) Specifies the name of the component to configure with the added content. The only allowable value is `Microsoft-Windows-Shell-Setup`.
* `setting_name` - (Required) Specifies the name of the setting to which the content applies. Possible values are: `FirstLogonCommands` and `AutoLogon`.
* `content` - (Optional) Specifies the base-64 encoded XML formatted content that is added to the unattend.xml file for the specified path and component.

`os_profile_linux_config` supports the following:

* `disable_password_authentication` - (Required) Specifies whether password authentication should be disabled.
* `ssh_keys` - (Optional) Specifies a collection of `path` and `key_data` to be placed on the virtual machine.

~> **Note:** Please note that the only allowed `path` is `/home/<username>/.ssh/authorized_keys` due to a limitation of Azure.

`os_profile_secrets` supports the following:

* `source_vault_id` - (Required) Specifies the key vault to use.
* `vault_certificates` - (Required) A collection of Vault Certificates as documented below

`vault_certificates` support the following:

* `certificate_url` - (Required) Specifies the URI of the key vault secrets in the format of `https://<vaultEndpoint>/secrets/<secretName>/<secretVersion>`. Stored secret is the Base64 encoding of a JSON Object that which is encoded in UTF-8 of which the contents need to be

```json
{ 
  "data":"<Base64-encoded-certificate>", 
  "dataType":"pfx",
  "password":"<pfx-file-password>" 
}
```

* `certificate_store` - (Required, on windows machines) Specifies the certificate store on the Virtual Machine where the certificate should be added to.

## Attributes Reference

The following attributes are exported:

* `id` - The virtual machine ID.

## Import

Virtual Machines can be imported using the `resource id`, e.g.

```hcl
terraform import azurerm_virtual_machine.test /subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/mygroup1/providers/microsoft.compute/virtualMachines/machine1
```
