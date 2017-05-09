# provider "azurerm" {
#   subscription_id = "REPLACE-WITH-YOUR-SUBSCRIPTION-ID"
#   client_id       = "REPLACE-WITH-YOUR-CLIENT-ID"
#   client_secret   = "REPLACE-WITH-YOUR-CLIENT-SECRET"
#   tenant_id       = "REPLACE-WITH-YOUR-TENANT-ID"
# }

resource "azurerm_resource_group" "rg" {
  name     = "${var.resource_group_name}"
  location = "${var.resource_group_location}"
}

resource "azurerm_key_vault" "quickstart" {
  name                = "${var.keyvault_name}"
  location            = "${azurerm_resource_group.quickstart.location}"
  resource_group_name = "${azurerm_resource_group.quickstart.name}"

  sku {
    name = "standard"
  }

  tenant_id = "${var.keyvault_tenant_id}"

  access_policy {
    tenant_id = "${var.keyvault_tenant_id}"
    object_id = "${var.keyvault_object_id}"

    key_permissions    = "${var.keys_permissions}"
    secret_permissions = "${var.secrets_permissions}"
  }

  enabled_for_deployment          = false
  enabled_for_disk_encryption     = false
  enabled_for_template_deployment = true
}
