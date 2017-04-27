resource "azurerm_resource_group" "rg" {
  name     = "${var.resource_group}"
  location = "${var.location}"
}

resource "azurerm_storage_account" "stor" {
  name                = "${var.hostname}stor"
  location            = "${var.location}"
  resource_group_name = "${azurerm_resource_group.rg.name}"
  account_type        = "${var.storage_account_type}"
}

resource "azurerm_cdn_profile" "cdn" {
  name                = "${var.hostname}CdnProfile1"
  location            = "${var.location}"
  resource_group_name = "${azurerm_resource_group.rg.name}"
  sku                 = "Standard_Akamai"
}

resource "azurerm_cdn_endpoint" "cdnendpt" {
  name                      = "${var.hostname}CdnEndpoint1"
  profile_name              = "${azurerm_cdn_profile.cdn.name}"
  location                  = "${var.location}"
  resource_group_name       = "${azurerm_resource_group.rg.name}"
  # content_types_to_compress = ["text/plain", "text/html", "text/css", "application/x-javascript", "text/javascript"]
  # is_compression_enabled    = true
  # is_https_allowed          = false

  origin {
    name       = "${var.hostname}Origin1"
    host_name  = "vmforcdn.southcentralus.cloudapp.azure.com"
    http_port  = 80
  }
}
