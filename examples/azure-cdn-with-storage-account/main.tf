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

resource "azurerm_storage_account" "stor" {
  name                = "${var.resource_group}stor"
  location            = "${var.location}"
  resource_group_name = "${azurerm_resource_group.rg.name}"
  account_type        = "${var.storage_account_type}"
}

resource "azurerm_cdn_profile" "cdn" {
  name                = "${var.resource_group}CdnProfile1"
  location            = "${var.location}"
  resource_group_name = "${azurerm_resource_group.rg.name}"
  sku                 = "Standard_Akamai"
}

resource "azurerm_cdn_endpoint" "cdnendpt" {
  name                      = "${var.resource_group}CdnEndpoint1"
  profile_name              = "${azurerm_cdn_profile.cdn.name}"
  location                  = "${var.location}"
  resource_group_name       = "${azurerm_resource_group.rg.name}"

  origin {
    name       = "${var.resource_group}Origin1"
    host_name  = "${var.host_name}"
    http_port  = 80
    https_port = 443
  }
}