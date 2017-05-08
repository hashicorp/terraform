#  "outputs": {
#     "BitLockerKey": {
#       "type": "string",
#       "value": "[reference(resourceId('Microsoft.Compute/virtualMachines/extensions',  parameters('vmName'), variables('extensionName'))).instanceView.statuses[0].message]"