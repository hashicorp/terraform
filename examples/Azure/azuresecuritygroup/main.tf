provider "azurerm" {
   subscription_id = "${var.subscriptionid}"
   client_id       = "${var.clientid}"
   client_secret   = "${var.clientsecret}"
   tenant_id       = "${var.tenantid}"
}

resource "azurerm_resource_group" "satya" {
  name     = "acceptanceTestResourceGroup1"
  location = "${var.location}"
}

resource "azurerm_network_security_group" "murali" {
  name                = "acceptanceTestSecurityGroup1"
  location            = "${azurerm_resource_group.satya.location}"
  resource_group_name = "${azurerm_resource_group.satya.name}"

  security_rule {
    name                       = "firewall"
    priority                   = 100
    direction                  = "Inbound"
    access                     = "Allow"
    protocol                   = "Tcp"
    source_port_range          = "*"
    destination_port_range     = "*"
    source_address_prefix      = "*"
    destination_address_prefix = "*"
  }

  tags {
    environment = "janasena"
  }
}
