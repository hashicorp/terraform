# 	"outputs": {
# 		"Openshift Console Url": {
# 			"type": "string",
# 			"value": "[concat('https://', reference(parameters('openshiftMasterPublicIpDnsLabel')).dnsSettings.fqdn, ':8443/console')]"
# 		},
# 		"Openshift Master SSH": {
# 			"type": "string",
# 			"value": "[concat('ssh ', parameters('adminUsername'), '@', reference(parameters('openshiftMasterPublicIpDnsLabel')).dnsSettings.fqdn, ' -p 2200')]"
# 		},
# 		"Openshift Infra Load Balancer FQDN": {
# 			"type": "string",
# 			"value": "[reference(parameters('infraLbPublicIpDnsLabel')).dnsSettings.fqdn]"
# 		},
# 		"Node OS Storage Account Name": {
# 			"type": "string",
# 			"value": "[variables('newStorageAccountNodeOs')]"
# 		},
# 		"Node Data Storage Account Name": {
# 			"type": "string",
# 			"value": "[variables('newStorageAccountNodeData')]"
# 		},
# 		"Infra Storage Account Name": {
# 			"type": "string",
# 			"value": "[variables('newStorageAccountInfra')]"
# 		}
# 	}
# }

