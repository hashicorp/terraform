resource "azurerm_resource_group" "rg" {
  name     = "${var.resource_group}"
  location = "${var.location}"
}

resource "azurerm_servicebus_namespace" "test" {
  depends_on          = ["azurerm_resource_group.rg"]
  name                = "${var.unique}ServiceBus"
  location            = "${var.location}"
  resource_group_name = "${var.resource_group}"
  sku                 = "standard"
}

resource "azurerm_servicebus_topic" "test" {
  name                = "${var.unique}Topic"
  location            = "${var.location}"
  resource_group_name = "${var.resource_group}"
  namespace_name      = "${azurerm_servicebus_namespace.test.name}"

  enable_partitioning = true
}

resource "azurerm_servicebus_subscription" "test" {
  name                = "${var.unique}Subscription"
  location            = "${var.location}"
  resource_group_name = "${var.resource_group}"
  namespace_name      = "${azurerm_servicebus_namespace.test.name}"
  topic_name          = "${azurerm_servicebus_topic.test.name}"
  max_delivery_count  = 1
}
