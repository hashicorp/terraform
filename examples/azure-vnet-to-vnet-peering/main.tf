# provider "azurerm" {
#   subscription_id = "REPLACE-WITH-YOUR-SUBSCRIPTION-ID"
#   client_id       = "REPLACE-WITH-YOUR-CLIENT-ID"
#   client_secret   = "REPLACE-WITH-YOUR-CLIENT-SECRET"
#   tenant_id       = "REPLACE-WITH-YOUR-TENANT-ID"
# }

resource "azurerm_resource_group" "rg" {
  name     = "${var.resource_group}"
  location = "${var.location}"
}

resource "azurerm_virtual_network" "vnet1" {
  name                = "${var.resource_group}-vnet1"
  location            = "${var.location}"
  address_space       = ["10.0.0.0/24"]
  resource_group_name = "${azurerm_resource_group.rg.name}"

  subnet {
    name           = "subnet1"
    address_prefix = "10.0.0.0/24"
  }
}

resource "azurerm_virtual_network" "vnet2" {
  name                = "${var.resource_group}-vnet2"
  location            = "${var.location}"
  address_space       = ["192.168.0.0/24"]
  resource_group_name = "${azurerm_resource_group.rg.name}"

  subnet {
    name           = "subnet1"
    address_prefix = "192.168.0.0/24"
  }
}

resource "azurerm_virtual_network_peering" "peer1" {
  name                         = "vNet1-to-vNet2"
  resource_group_name          = "${azurerm_resource_group.rg.name}"
  virtual_network_name         = "${azurerm_virtual_network.vnet1.name}"
  remote_virtual_network_id    = "${azurerm_virtual_network.vnet2.id}"
  allow_virtual_network_access = true
  allow_forwarded_traffic      = false
  allow_gateway_transit        = false
}

resource "azurerm_virtual_network_peering" "peer2" {
  name                         = "vNet2-to-vNet1"
  resource_group_name          = "${azurerm_resource_group.rg.name}"
  virtual_network_name         = "${azurerm_virtual_network.vnet2.name}"
  remote_virtual_network_id    = "${azurerm_virtual_network.vnet1.id}"
  allow_virtual_network_access = true
  allow_forwarded_traffic      = false
  allow_gateway_transit        = false
  use_remote_gateways          = false
}
