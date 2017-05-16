# output "vault_uri" {
#   value = ["${azurerm_key_vault.vault.vault_uri}"]
# }
#  "outputs": {
#     "BitLockerKey": {
#       "type": "string",
#       "value": "[reference(resourceId('Microsoft.Compute/virtualMachines/extensions',  parameters('vmName'), variables('extensionName'))).instanceView.statuses[0].message]"

