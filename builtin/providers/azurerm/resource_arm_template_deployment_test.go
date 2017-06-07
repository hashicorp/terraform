package azurerm

import (
	"fmt"
	"net/http"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform/helper/acctest"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

func TestAccAzureRMTemplateDeployment_basic(t *testing.T) {
	ri := acctest.RandInt()
	config := fmt.Sprintf(testAccAzureRMTemplateDeployment_basicMultiple, ri, ri)
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testCheckAzureRMTemplateDeploymentDestroy,
		Steps: []resource.TestStep{
			{
				Config: config,
				Check: resource.ComposeTestCheckFunc(
					testCheckAzureRMTemplateDeploymentExists("azurerm_template_deployment.test"),
				),
			},
		},
	})
}

func TestAccAzureRMTemplateDeployment_disappears(t *testing.T) {
	ri := acctest.RandInt()
	config := fmt.Sprintf(testAccAzureRMTemplateDeployment_basicSingle, ri, ri, ri)
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testCheckAzureRMTemplateDeploymentDestroy,
		Steps: []resource.TestStep{
			{
				Config: config,
				Check: resource.ComposeTestCheckFunc(
					testCheckAzureRMTemplateDeploymentExists("azurerm_template_deployment.test"),
					testCheckAzureRMTemplateDeploymentDisappears("azurerm_template_deployment.test"),
				),
				ExpectNonEmptyPlan: true,
			},
		},
	})
}

func TestAccAzureRMTemplateDeployment_withParams(t *testing.T) {
	ri := acctest.RandInt()
	config := fmt.Sprintf(testAccAzureRMTemplateDeployment_withParams, ri, ri, ri)
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testCheckAzureRMTemplateDeploymentDestroy,
		Steps: []resource.TestStep{
			{
				Config: config,
				Check: resource.ComposeTestCheckFunc(
					testCheckAzureRMTemplateDeploymentExists("azurerm_template_deployment.test"),
					resource.TestCheckResourceAttr("azurerm_template_deployment.test", "outputs.testOutput", "Output Value"),
				),
			},
		},
	})
}

func TestAccAzureRMTemplateDeployment_withOutputs(t *testing.T) {
	ri := acctest.RandInt()
	config := fmt.Sprintf(testAccAzureRMTemplateDeployment_withOutputs, ri, ri, ri)
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testCheckAzureRMTemplateDeploymentDestroy,
		Steps: []resource.TestStep{
			{
				Config: config,
				Check: resource.ComposeTestCheckFunc(
					testCheckAzureRMTemplateDeploymentExists("azurerm_template_deployment.test"),
					resource.TestCheckOutput("tfIntOutput", "-123"),
					resource.TestCheckOutput("tfStringOutput", "Standard_GRS"),
					resource.TestCheckOutput("tfFalseOutput", "false"),
					resource.TestCheckOutput("tfTrueOutput", "true"),
					resource.TestCheckResourceAttr("azurerm_template_deployment.test", "outputs.stringOutput", "Standard_GRS"),
				),
			},
		},
	})
}

func TestAccAzureRMTemplateDeployment_withError(t *testing.T) {
	ri := acctest.RandInt()
	config := fmt.Sprintf(testAccAzureRMTemplateDeployment_withError, ri, ri)
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testCheckAzureRMTemplateDeploymentDestroy,
		Steps: []resource.TestStep{
			{
				Config:      config,
				ExpectError: regexp.MustCompile("The deployment operation failed"),
			},
		},
	})
}

func testCheckAzureRMTemplateDeploymentExists(name string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		// Ensure we have enough information in state to look up in API
		rs, ok := s.RootModule().Resources[name]
		if !ok {
			return fmt.Errorf("Not found: %s", name)
		}

		name := rs.Primary.Attributes["name"]
		resourceGroup, hasResourceGroup := rs.Primary.Attributes["resource_group_name"]
		if !hasResourceGroup {
			return fmt.Errorf("Bad: no resource group found in state for template deployment: %s", name)
		}

		conn := testAccProvider.Meta().(*ArmClient).deploymentsClient

		resp, err := conn.Get(resourceGroup, name)
		if err != nil {
			return fmt.Errorf("Bad: Get on deploymentsClient: %s", err)
		}

		if resp.StatusCode == http.StatusNotFound {
			return fmt.Errorf("Bad: TemplateDeployment %q (resource group: %q) does not exist", name, resourceGroup)
		}

		return nil
	}
}

func testCheckAzureRMTemplateDeploymentDisappears(name string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		// Ensure we have enough information in state to look up in API
		rs, ok := s.RootModule().Resources[name]
		if !ok {
			return fmt.Errorf("Not found: %s", name)
		}

		name := rs.Primary.Attributes["name"]
		resourceGroup, hasResourceGroup := rs.Primary.Attributes["resource_group_name"]
		if !hasResourceGroup {
			return fmt.Errorf("Bad: no resource group found in state for template deployment: %s", name)
		}

		conn := testAccProvider.Meta().(*ArmClient).deploymentsClient

		_, error := conn.Delete(resourceGroup, name, make(chan struct{}))
		err := <-error
		if err != nil {
			return fmt.Errorf("Bad: Delete on deploymentsClient: %s", err)
		}

		return nil
	}
}

func testCheckAzureRMTemplateDeploymentDestroy(s *terraform.State) error {
	conn := testAccProvider.Meta().(*ArmClient).vmClient

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "azurerm_template_deployment" {
			continue
		}

		name := rs.Primary.Attributes["name"]
		resourceGroup := rs.Primary.Attributes["resource_group_name"]

		resp, err := conn.Get(resourceGroup, name, "")

		if err != nil {
			return nil
		}

		if resp.StatusCode != http.StatusNotFound {
			return fmt.Errorf("Template Deployment still exists:\n%#v", resp.VirtualMachineProperties)
		}
	}

	return nil
}

var testAccAzureRMTemplateDeployment_basicSingle = `
  resource "azurerm_resource_group" "test" {
    name = "acctestRG-%d"
    location = "West US"
  }

  resource "azurerm_template_deployment" "test" {
    name = "acctesttemplate-%d"
    resource_group_name = "${azurerm_resource_group.test.name}"
    template_body = <<DEPLOY
{
  "$schema": "https://schema.management.azure.com/schemas/2015-01-01/deploymentTemplate.json#",
  "contentVersion": "1.0.0.0",
  "variables": {
    "location": "[resourceGroup().location]",
    "publicIPAddressType": "Dynamic",
    "apiVersion": "2015-06-15",
    "dnsLabelPrefix": "[concat('terraform-tdacctest', uniquestring(resourceGroup().id))]"
  },
  "resources": [
     {
      "type": "Microsoft.Network/publicIPAddresses",
      "apiVersion": "[variables('apiVersion')]",
      "name": "acctestpip-%d",
      "location": "[variables('location')]",
      "properties": {
        "publicIPAllocationMethod": "[variables('publicIPAddressType')]",
        "dnsSettings": {
          "domainNameLabel": "[variables('dnsLabelPrefix')]"
        }
      }
    }
  ]
}
DEPLOY
    deployment_mode = "Complete"
  }

`

var testAccAzureRMTemplateDeployment_basicMultiple = `
  resource "azurerm_resource_group" "test" {
    name = "acctestRG-%d"
    location = "West US"
  }

  resource "azurerm_template_deployment" "test" {
    name = "acctesttemplate-%d"
    resource_group_name = "${azurerm_resource_group.test.name}"
    template_body = <<DEPLOY
{
  "$schema": "https://schema.management.azure.com/schemas/2015-01-01/deploymentTemplate.json#",
  "contentVersion": "1.0.0.0",
  "parameters": {
    "storageAccountType": {
      "type": "string",
      "defaultValue": "Standard_LRS",
      "allowedValues": [
        "Standard_LRS",
        "Standard_GRS",
        "Standard_ZRS"
      ],
      "metadata": {
        "description": "Storage Account type"
      }
    }
  },
  "variables": {
    "location": "[resourceGroup().location]",
    "storageAccountName": "[concat(uniquestring(resourceGroup().id), 'storage')]",
    "publicIPAddressName": "[concat('myPublicIp', uniquestring(resourceGroup().id))]",
    "publicIPAddressType": "Dynamic",
    "apiVersion": "2015-06-15",
    "dnsLabelPrefix": "[concat('terraform-tdacctest', uniquestring(resourceGroup().id))]"
  },
  "resources": [
    {
      "type": "Microsoft.Storage/storageAccounts",
      "name": "[variables('storageAccountName')]",
      "apiVersion": "[variables('apiVersion')]",
      "location": "[variables('location')]",
      "properties": {
        "accountType": "[parameters('storageAccountType')]"
      }
    },
    {
      "type": "Microsoft.Network/publicIPAddresses",
      "apiVersion": "[variables('apiVersion')]",
      "name": "[variables('publicIPAddressName')]",
      "location": "[variables('location')]",
      "properties": {
        "publicIPAllocationMethod": "[variables('publicIPAddressType')]",
        "dnsSettings": {
          "domainNameLabel": "[variables('dnsLabelPrefix')]"
        }
      }
    }
  ]
}
DEPLOY
    deployment_mode = "Complete"
  }

`

var testAccAzureRMTemplateDeployment_withParams = `
  resource "azurerm_resource_group" "test" {
    name = "acctestRG-%d"
    location = "West US"
  }

  output "test" {
    value = "${azurerm_template_deployment.test.outputs["testOutput"]}"
  }

  resource "azurerm_storage_container" "using-outputs" {
    name = "vhds"
    resource_group_name = "${azurerm_resource_group.test.name}"
    storage_account_name = "${azurerm_template_deployment.test.outputs["accountName"]}"
    container_access_type = "private"
  }

  resource "azurerm_template_deployment" "test" {
    name = "acctesttemplate-%d"
    resource_group_name = "${azurerm_resource_group.test.name}"
    template_body = <<DEPLOY
{
  "$schema": "https://schema.management.azure.com/schemas/2015-01-01/deploymentTemplate.json#",
  "contentVersion": "1.0.0.0",
  "parameters": {
    "storageAccountType": {
      "type": "string",
      "defaultValue": "Standard_LRS",
      "allowedValues": [
        "Standard_LRS",
        "Standard_GRS",
        "Standard_ZRS"
      ],
      "metadata": {
        "description": "Storage Account type"
      }
    },
    "dnsLabelPrefix": {
      "type": "string",
      "metadata": {
        "description": "DNS Label for the Public IP. Must be lowercase. It should match with the following regular expression: ^[a-z][a-z0-9-]{1,61}[a-z0-9]$ or it will raise an error."
      }
    }
  },
  "variables": {
    "location": "[resourceGroup().location]",
    "storageAccountName": "[concat(uniquestring(resourceGroup().id), 'storage')]",
    "publicIPAddressName": "[concat('myPublicIp', uniquestring(resourceGroup().id))]",
    "publicIPAddressType": "Dynamic",
    "apiVersion": "2015-06-15"
  },
  "resources": [
    {
      "type": "Microsoft.Storage/storageAccounts",
      "name": "[variables('storageAccountName')]",
      "apiVersion": "[variables('apiVersion')]",
      "location": "[variables('location')]",
      "properties": {
        "accountType": "[parameters('storageAccountType')]"
      }
    },
    {
      "type": "Microsoft.Network/publicIPAddresses",
      "apiVersion": "[variables('apiVersion')]",
      "name": "[variables('publicIPAddressName')]",
      "location": "[variables('location')]",
      "properties": {
        "publicIPAllocationMethod": "[variables('publicIPAddressType')]",
        "dnsSettings": {
          "domainNameLabel": "[parameters('dnsLabelPrefix')]"
        }
      }
    }
  ],
  "outputs": {
    "testOutput": {
      "type": "string",
      "value": "Output Value"
    },
    "accountName": {
      "type": "string",
      "value": "[variables('storageAccountName')]"
    }
  }
}
DEPLOY
    parameters {
	dnsLabelPrefix = "terraform-test-%d"
	storageAccountType = "Standard_GRS"
    }
    deployment_mode = "Complete"
  }

`

var testAccAzureRMTemplateDeployment_withOutputs = `
  resource "azurerm_resource_group" "test" {
    name = "acctestRG-%d"
    location = "West US"
  }

  output "tfStringOutput" {
    value = "${azurerm_template_deployment.test.outputs.stringOutput}"
  }

  output "tfIntOutput" {
    value = "${azurerm_template_deployment.test.outputs.intOutput}"
  }

  output "tfFalseOutput" {
    value = "${azurerm_template_deployment.test.outputs.falseOutput}"
  }

  output "tfTrueOutput" {
    value = "${azurerm_template_deployment.test.outputs.trueOutput}"
  }

  resource "azurerm_template_deployment" "test" {
    name = "acctesttemplate-%d"
    resource_group_name = "${azurerm_resource_group.test.name}"
    template_body = <<DEPLOY
{
  "$schema": "https://schema.management.azure.com/schemas/2015-01-01/deploymentTemplate.json#",
  "contentVersion": "1.0.0.0",
  "parameters": {
    "storageAccountType": {
      "type": "string",
      "defaultValue": "Standard_LRS",
      "allowedValues": [
        "Standard_LRS",
        "Standard_GRS",
        "Standard_ZRS"
      ],
      "metadata": {
        "description": "Storage Account type"
      }
    },
    "dnsLabelPrefix": {
      "type": "string",
      "metadata": {
        "description": "DNS Label for the Public IP. Must be lowercase. It should match with the following regular expression: ^[a-z][a-z0-9-]{1,61}[a-z0-9]$ or it will raise an error."
      }
    },
    "intParameter": {
      "type": "int",
      "defaultValue": -123
    },
    "falseParameter": {
      "type": "bool",
      "defaultValue": false
    },
    "trueParameter": {
      "type": "bool",
      "defaultValue": true
    }
  },
  "variables": {
    "location": "[resourceGroup().location]",
    "storageAccountName": "[concat(uniquestring(resourceGroup().id), 'storage')]",
    "publicIPAddressName": "[concat('myPublicIp', uniquestring(resourceGroup().id))]",
    "publicIPAddressType": "Dynamic",
    "apiVersion": "2015-06-15"
  },
  "resources": [
    {
      "type": "Microsoft.Storage/storageAccounts",
      "name": "[variables('storageAccountName')]",
      "apiVersion": "[variables('apiVersion')]",
      "location": "[variables('location')]",
      "properties": {
        "accountType": "[parameters('storageAccountType')]"
      }
    },
    {
      "type": "Microsoft.Network/publicIPAddresses",
      "apiVersion": "[variables('apiVersion')]",
      "name": "[variables('publicIPAddressName')]",
      "location": "[variables('location')]",
      "properties": {
        "publicIPAllocationMethod": "[variables('publicIPAddressType')]",
        "dnsSettings": {
          "domainNameLabel": "[parameters('dnsLabelPrefix')]"
        }
      }
    }
  ],
  "outputs": {
    "stringOutput": {
      "type": "string",
      "value": "[parameters('storageAccountType')]"
    },
    "intOutput": {
      "type": "int",
      "value": "[parameters('intParameter')]"
    },
    "falseOutput": {
      "type": "bool",
      "value": "[parameters('falseParameter')]"
    },
    "trueOutput": {
      "type": "bool",
      "value": "[parameters('trueParameter')]"
    }
  }
}
DEPLOY
    parameters {
      dnsLabelPrefix = "terraform-test-%d"
      storageAccountType = "Standard_GRS"
    }
    deployment_mode = "Incremental"
  }

`

// StorageAccount name is too long, forces error
var testAccAzureRMTemplateDeployment_withError = `
  resource "azurerm_resource_group" "test" {
    name = "acctestRG-%d"
    location = "West US"
  }

  output "test" {
    value = "${azurerm_template_deployment.test.outputs.testOutput}"
  }

  resource "azurerm_template_deployment" "test" {
    name = "acctesttemplate-%d"
    resource_group_name = "${azurerm_resource_group.test.name}"
    template_body = <<DEPLOY
{
  "$schema": "https://schema.management.azure.com/schemas/2015-01-01/deploymentTemplate.json#",
  "contentVersion": "1.0.0.0",
  "parameters": {
    "storageAccountType": {
      "type": "string",
      "defaultValue": "Standard_LRS",
      "allowedValues": [
        "Standard_LRS",
        "Standard_GRS",
        "Standard_ZRS"
      ],
      "metadata": {
        "description": "Storage Account type"
      }
    }
  },
  "variables": {
    "location": "[resourceGroup().location]",
    "storageAccountName": "badStorageAccountNameTooLong",
    "apiVersion": "2015-06-15"
  },
  "resources": [
    {
      "type": "Microsoft.Storage/storageAccounts",
      "name": "[variables('storageAccountName')]",
      "apiVersion": "[variables('apiVersion')]",
      "location": "[variables('location')]",
      "properties": {
        "accountType": "[parameters('storageAccountType')]"
      }
    }
  ],
  "outputs": {
    "testOutput": {
      "type": "string",
      "value": "Output Value"
    }
  }
}
DEPLOY
    parameters {
        storageAccountType = "Standard_GRS"
    }
    deployment_mode = "Complete"
  }
`
