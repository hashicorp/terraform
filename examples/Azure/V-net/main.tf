 Configure the Azure Provider

 
provider "azurerm" {
   subscription_id = "${var.subscriptionid}"
   client_id       = "${var.clientid}"
   client_secret   = "${var.clientsecret}"
   tenant_id       = "${var.tenantid}"
}

provider "azurerm" {}

# Create a resource group
resource "azurerm_resource_group" "satya" {
  name     = "terraform"
  location = "${var.location}"
}

# Create a virtual network within the resource group
resource "azurerm_virtual_network" "muralidhar" {
  name                = "terraform-network"
  address_space       = ["192.168.10.0/27"]
  location            = "${var.location}"
  resource_group_name = "${azurerm_resource_group.satya.name}"

  subnet {
    name           = "subnet1"
    address_prefix = "192.168.10.0/28"
  }

  subnet {
    name           = "subnet2"
    address_prefix = "192.168.10.32/29"
  }

  subnet {
    name           = "subnet3"
    address_prefix = "192.168.10.48/29"
  }
}
