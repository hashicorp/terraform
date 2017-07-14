# ********************** WEBSITE DEPLOYMENT TEMPLATE ********************** #

resource "azurerm_template_deployment" "website" {
  name                = "website"
  resource_group_name = "${azurerm_resource_group.rg.name}"
  depends_on          = ["azurerm_virtual_machine_extension.setup_mysql"]

  template_body = <<DEPLOY
{  
   "$schema":"http://schema.management.azure.com/schemas/2014-04-01-preview/deploymentTemplate.json#",
   "contentVersion":"1.0.0.0",
   "parameters":{  
      "siteName":{  
         "type":"string",
         "defaultValue":"${var.site_name}"
      },
      "hostingPlanName":{  
         "type":"string",
         "defaultValue":"${var.hosting_plan_name}"
      },
      "sku":{  
         "type":"string",
         "allowedValues":[  
            "Free",
            "Shared",
            "Basic",
            "Standard",
            "Premium"
         ],
         "defaultValue":"${var.sku}"
      },
      "workerSize":{  
         "type":"string",
         "allowedValues":[  
            "0",
            "1",
            "2"
         ],
         "defaultValue":"${var.worker_size}"
      },
      "dbServer":{  
         "type":"string",
         "defaultValue":"${var.dns_name}.${azurerm_resource_group.rg.location}.cloudapp.azure.com:${var.mysql_front_end_port_0}"
      },
      "dbName":{  
         "type":"string",
         "defaultValue":"${var.unique_prefix}wordpress"
      },
      "dbAdminPassword":{  
         "type":"string",
         "defaultValue":"${var.mysql_root_password}"
      }
   },
   "variables":{  
      "connectionString":"[concat('Database=', parameters('dbName'), ';Data Source=', parameters('dbServer'), ';User Id=admin;Password=', parameters('dbAdminPassword'))]",
      "repoUrl":"https://github.com/azureappserviceoss/wordpress-azure",
      "branch":"master",
      "workerSize":"[parameters('workerSize')]",
      "sku":"[parameters('sku')]",
      "hostingPlanName":"[parameters('hostingPlanName')]"
   },
   "resources":[  
      {  
         "apiVersion":"2014-06-01",
         "name":"[variables('hostingPlanName')]",
         "type":"Microsoft.Web/serverfarms",
         "location":"[resourceGroup().location]",
         "properties":{  
            "name":"[variables('hostingPlanName')]",
            "sku":"[variables('sku')]",
            "workerSize":"[variables('workerSize')]",
            "hostingEnvironment":"",
            "numberOfWorkers":0
         }
      },
      {  
         "apiVersion":"2015-02-01",
         "name":"[parameters('siteName')]",
         "type":"Microsoft.Web/sites",
         "location":"[resourceGroup().location]",
         "tags":{  
            "[concat('hidden-related:', '/subscriptions/', subscription().subscriptionId,'/resourcegroups/', resourceGroup().name, '/providers/Microsoft.Web/serverfarms/', variables('hostingPlanName'))]":"empty"
         },
         "dependsOn":[  
            "[concat('Microsoft.Web/serverfarms/', variables('hostingPlanName'))]"
         ],
         "properties":{  
            "name":"[parameters('siteName')]",
            "serverFarmId":"[concat('/subscriptions/', subscription().subscriptionId,'/resourcegroups/', resourceGroup().name, '/providers/Microsoft.Web/serverfarms/', variables('hostingPlanName'))]",
            "hostingEnvironment":""
         },
         "resources":[  
            {  
               "apiVersion":"2015-04-01",
               "name":"connectionstrings",
               "type":"config",
               "dependsOn":[  
                  "[concat('Microsoft.Web/Sites/', parameters('siteName'))]"
               ],
               "properties":{  
                  "defaultConnection":{  
                     "value":"[variables('connectionString')]",
                     "type":"MySQL"
                  }
               }
            },
            {  
               "apiVersion":"2015-04-01",
               "name":"web",
               "type":"config",
               "dependsOn":[  
                  "[concat('Microsoft.Web/Sites/', parameters('siteName'))]"
               ],
               "properties":{  
                  "phpVersion":"5.6"
               }
            },
            {  
               "apiVersion":"2015-08-01",
               "name":"web",
               "type":"sourcecontrols",
               "dependsOn":[  
                  "[resourceId('Microsoft.Web/Sites', parameters('siteName'))]",
                  "[concat('Microsoft.Web/Sites/', parameters('siteName'), '/config/connectionstrings')]",
                  "[concat('Microsoft.Web/Sites/', parameters('siteName'), '/config/web')]"
               ],
               "properties":{  
                  "RepoUrl":"[variables('repoUrl')]",
                  "branch":"[variables('branch')]",
                  "IsManualIntegration":true
               }
            }
         ]
      }      
   ]
}
DEPLOY

  deployment_mode = "Incremental"
}
