resource "azurerm_resource_group" "rg" {
  name     = "${var.resource_group}"
  location = "${var.location}"
}

resource "azurerm_public_ip" "pip" {
  name                         = "ip${count.index}"
  location                     = "${var.location}"
  resource_group_name          = "${azurerm_resource_group.rg.name}"
  public_ip_address_allocation = "dynamic"
  domain_name_label            = "${var.dns_name}${count.index}"
  count                        = "${var.num_vms}"
}

resource "azurerm_virtual_network" "vnet" {
  name                = "${var.virtual_network_name}"
  location            = "${var.location}"
  address_space       = ["${var.address_space}"]
  resource_group_name = "${azurerm_resource_group.rg.name}"
}

resource "azurerm_subnet" "subnet" {
  name                 = "${var.rg_prefix}subnet"
  virtual_network_name = "${azurerm_virtual_network.vnet.name}"
  resource_group_name  = "${azurerm_resource_group.rg.name}"
  address_prefix       = "${var.subnet_prefix}"
}

resource "azurerm_network_interface" "nic" {
  name                = "nic${count.index}"
  location            = "${var.location}"
  resource_group_name = "${azurerm_resource_group.rg.name}"
  count               = "${var.num_vms}"

  ip_configuration {
    name                          = "ipconfig${count.index}"
    subnet_id                     = "${azurerm_subnet.subnet.id}"
    private_ip_address_allocation = "Dynamic"
    public_ip_address_id          = ["${element(azurerm_public_ip.pip.*.id, count.index)}"]
  }
}

resource "azurerm_storage_account" "stor" {
  name                = "${var.dns_name}stor"
  location            = "${var.location}"
  resource_group_name = "${azurerm_resource_group.rg.name}"
  account_type        = "${var.storage_account_type}"
}

resource "azurerm_virtual_machine" "vm" {
  name                  = "vm${count.index}"
  location              = "${var.location}"
  resource_group_name   = "${azurerm_resource_group.rg.name}"
  vm_size               = "${var.vm_size}"
  network_interface_ids = ["${element(azurerm_network_interface.nic.*.id, count.index)}"]

  storage_image_reference {
    publisher = "${var.image_publisher}"
    offer     = "${var.image_offer}"
    sku       = "${var.image_sku}"
    version   = "${var.image_version}"
  }

  storage_os_disk {
    name              = "osdisk${count.index}"
    create_option     = "FromImage"
  }

  os_profile {
    computer_name  = "${var.hostname}"
    admin_username = "${var.admin_username}"
    admin_password = "${var.admin_password}"
  }

  os_profile_linux_config {
    disable_password_authentication = false
  }

  boot_diagnostics {
    enabled     = "true"
    storage_uri = "${azurerm_storage_account.stor.primary_blob_endpoint}"
  }
}

      "type": "Microsoft.Compute/virtualMachines/extensions",
      "name": "[concat(variables('vmName'), copyIndex(), '/installcustomscript')]",
      "copy": {
        "name": "extLoop",
        "count": "[variables('numVMs')]"
      },
      "apiVersion": "[variables('apiVersion')]",
      "location": "[resourceGroup().location]",
      "dependsOn": [
        "[concat('Microsoft.Compute/virtualMachines/', variables('vmName'), copyIndex())]"
      ],
      "properties": {
        "publisher": "Microsoft.Azure.Extensions",
        "type": "CustomScript",
        "typeHandlerVersion": "2.0",
        "autoUpgradeMinorVersion": true,
        "settings": {
          "commandToExecute": "sudo bash -c 'apt-get update && apt-get -y install apache2' "
        }
      }
    },
    {
      "apiVersion": "[variables('tmApiVersion')]",
      "type": "Microsoft.Network/trafficManagerProfiles",
      "name": "VMEndpointExample",
      "location": "global",
      "dependsOn": [
        "[concat('Microsoft.Network/publicIPAddresses/', variables('publicIPAddressName'), '0')]",
        "[concat('Microsoft.Network/publicIPAddresses/', variables('publicIPAddressName'), '1')]",
        "[concat('Microsoft.Network/publicIPAddresses/', variables('publicIPAddressName'), '2')]"
      ],
      "properties": {
        "profileStatus": "Enabled",
        "trafficRoutingMethod": "Weighted",
        "dnsConfig": {
          "relativeName": "[parameters('uniqueDnsName')]",
          "ttl": 30
        },
        "monitorConfig": {
          "protocol": "http",
          "port": 80,
          "path": "/"
        },
        "endpoints": [
          {
            "name": "endpoint0",
            "type": "Microsoft.Network/trafficManagerProfiles/azureEndpoints",
            "properties": {
              "targetResourceId": "[resourceId('Microsoft.Network/publicIPAddresses',concat(variables('publicIPAddressName'), 0))]",
              "endpointStatus": "Enabled",
              "weight": 1
            }
          },
          {
            "name": "endpoint1",
            "type": "Microsoft.Network/trafficManagerProfiles/azureEndpoints",
            "properties": {
              "targetResourceId": "[resourceId('Microsoft.Network/publicIPAddresses',concat(variables('publicIPAddressName'), 1))]",
              "endpointStatus": "Enabled",
              "weight": 1
            }
          },
          {
            "name": "endpoint2",
            "type": "Microsoft.Network/trafficManagerProfiles/azureEndpoints",
            "properties": {
              "targetResourceId": "[resourceId('Microsoft.Network/publicIPAddresses',concat(variables('publicIPAddressName'), 2))]",
              "endpointStatus": "Enabled",
              "weight": 1
            }
          }
        ]
      }
    }
  ]
}