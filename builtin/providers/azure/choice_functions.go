package azure

import (
	"github.com/hashicorp/terraform/helper/schema"
)

// resourceAzureInstanceCreateChoiceFunc goes ahead and chooses which implementation of
// the Create function should be used for creating the Azure instance.
func resourceAzureInstanceCreateChoiceFunc(d *schema.ResourceData, meta interface{}) error {
	if d.Get("use_asm_api").(bool) {
		return resourceAsmInstanceCreate(d, meta)
	}

	return resourceArmInstanceCreate(d, meta)
}

// resourceAzureInstanceReadChoiceFunc goes ahead and chooses which implementation of
// the Read function should be used for reading the Azure instance.
func resourceAzureInstanceReadChoiceFunc(d *schema.ResourceData, meta interface{}) error {
	if d.Get("use_asm_api").(bool) {
		return resourceAsmInstanceRead(d, meta)
	}

	return resourceArmInstanceRead(d, meta)
}

// resourceAzureInstanceUpdateChoiceFunc goes ahead and chooses which implementation of
// the Update function should be used for updating the Azure instance.
func resourceAzureInstanceUpdateChoiceFunc(d *schema.ResourceData, meta interface{}) error {
	if d.Get("use_asm_api").(bool) {
		return resourceAsmInstanceUpdate(d, meta)
	}

	return resourceArmInstanceUpdate(d, meta)
}

// resourceAzureInstanceDeleteChoiceFunc goes ahead and chooses which implementation of
// the Delete function should be used for deleting the Azure instance.
func resourceAzureInstanceDeleteChoiceFunc(d *schema.ResourceData, meta interface{}) error {
	if d.Get("use_asm_api").(bool) {
		return resourceAsmInstanceDelete(d, meta)
	}

	return resourceArmInstanceDelete(d, meta)
}

// resourceAzureDnsServerCreateChoiceFunc goes ahead and chooses which implementation of
// the Create function should be used for creating the Azure dns server.
func resourceAzureDnsServerCreateChoiceFunc(d *schema.ResourceData, meta interface{}) error {
	if d.Get("use_asm_api").(bool) {
		return resourceAsmDnsServerCreate(d, meta)
	}

	return resourceArmDnsServerCreate(d, meta)
}

// resourceAzureDnsServerReadChoiceFunc goes ahead and chooses which implementation of
// the Read function should be used for reading the Azure dns server.
func resourceAzureDnsServerReadChoiceFunc(d *schema.ResourceData, meta interface{}) error {
	if d.Get("use_asm_api").(bool) {
		return resourceAsmDnsServerRead(d, meta)
	}

	return resourceArmDnsServerRead(d, meta)
}

// resourceAzureDnsServerUpdateChoiceFunc goes ahead and chooses which implementation of
// the Update function should be used for updating the Azure dns server.
func resourceAzureDnsServerUpdateChoiceFunc(d *schema.ResourceData, meta interface{}) error {
	if d.Get("use_asm_api").(bool) {
		return resourceAsmDnsServerUpdate(d, meta)
	}

	return resourceArmDnsServerUpdate(d, meta)
}

// resourceAzureDnsServerExistsChoiceFunc goes ahead and chooses which implementation of
// the Exists function should be used for checking for the existence of the Azure dns server.
func resourceAzureDnsServerExistsChoiceFunc(d *schema.ResourceData, meta interface{}) (bool, error) {
	if d.Get("use_asm_api").(bool) {
		return resourceAsmDnsServerExists(d, meta)
	}

	return resourceArmDnsServerExists(d, meta)
}

// resourceAzureDnsServerDeleteChoiceFunc goes ahead and chooses which implementation of
// the Delete function should be used for deleting the Azure dns server.
func resourceAzureDnsServerDeleteChoiceFunc(d *schema.ResourceData, meta interface{}) error {
	if d.Get("use_asm_api").(bool) {
		return resourceAsmDnsServerDelete(d, meta)
	}

	return resourceArmDnsServerDelete(d, meta)
}

// resourceAzureLocalNetworkConnectionCreateChoiceFunc goes ahead and chooses which implementation of
// the Create function should be used for creating the Azure local network connection.
func resourceAzureLocalNetworkConnectionCreateChoiceFunc(d *schema.ResourceData, meta interface{}) error {
	if d.Get("use_asm_api").(bool) {
		return resourceAsmLocalNetworkConnectionCreate(d, meta)
	}

	return resourceArmLocalNetworkConnectionCreate(d, meta)
}

// resourceAzureLocalNetworkConnectionReadChoiceFunc goes ahead and chooses which implementation of
// the Read function should be used for reading the Azure local network connection.
func resourceAzureLocalNetworkConnectionReadChoiceFunc(d *schema.ResourceData, meta interface{}) error {
	if d.Get("use_asm_api").(bool) {
		return resourceAsmLocalNetworkConnectionRead(d, meta)
	}

	return resourceArmLocalNetworkConnectionRead(d, meta)
}

// resourceAzureLocalNetworkConnectionUpdateChoiceFunc goes ahead and chooses which implementation of
// the Update function should be used for updating the Azure local network connection.
func resourceAzureLocalNetworkConnectionUpdateChoiceFunc(d *schema.ResourceData, meta interface{}) error {
	if d.Get("use_asm_api").(bool) {
		return resourceAsmLocalNetworkConnectionUpdate(d, meta)
	}

	return resourceArmLocalNetworkConnectionUpdate(d, meta)
}

// resourceAzureLocalNetworkConnectionExistsChoiceFunc goes ahead and chooses which implementation of
// the Exists function should be used for checking for the existence of the Azure local network connection.
func resourceAzureLocalNetworkConnectionExistsChoiceFunc(d *schema.ResourceData, meta interface{}) (bool, error) {
	if d.Get("use_asm_api").(bool) {
		return resourceAsmLocalNetworkConnectionExists(d, meta)
	}

	return resourceArmLocalNetworkConnectionExists(d, meta)
}

// resourceAzureLocalNetworkConnectionDeleteChoiceFunc goes ahead and chooses which implementation of
// the Delete function should be used for deleting the Azure local network connection.
func resourceAzureLocalNetworkConnectionDeleteChoiceFunc(d *schema.ResourceData, meta interface{}) error {
	if d.Get("use_asm_api").(bool) {
		return resourceAsmLocalNetworkConnectionDelete(d, meta)
	}

	return resourceArmLocalNetworkConnectionDelete(d, meta)
}

// resourceAzureSecurityGroupCreateChoiceFunc goes ahead and chooses which implementation of
// the Create function should be used for creating the Azure security group.
func resourceAzureSecurityGroupCreateChoiceFunc(d *schema.ResourceData, meta interface{}) error {
	if d.Get("use_asm_api").(bool) {
		return resourceAsmSecurityGroupCreate(d, meta)
	}

	return resourceArmSecurityGroupCreate(d, meta)
}

// resourceAzureSecurityGroupReadChoiceFunc goes ahead and chooses which implementation of
// the Read function should be used for reading the Azure security group.
func resourceAzureSecurityGroupReadChoiceFunc(d *schema.ResourceData, meta interface{}) error {
	if d.Get("use_asm_api").(bool) {
		return resourceAsmSecurityGroupRead(d, meta)
	}

	return resourceArmSecurityGroupRead(d, meta)
}

// resourceAzureSecurityGroupDeleteChoiceFunc goes ahead and chooses which implementation of
// the Delete function should be used for deleting the Azure security group.
func resourceAzureSecurityGroupDeleteChoiceFunc(d *schema.ResourceData, meta interface{}) error {
	if d.Get("use_asm_api").(bool) {
		return resourceAsmSecurityGroupDelete(d, meta)
	}

	return resourceArmSecurityGroupDelete(d, meta)
}

// resourceAzureSecurityGroupRuleCreateChoiceFunc goes ahead and chooses which implementation of
// the Create function should be used for creating the Azure security group rule.
func resourceAzureSecurityGroupRuleCreateChoiceFunc(d *schema.ResourceData, meta interface{}) error {
	if d.Get("use_asm_api").(bool) {
		return resourceAsmSecurityGroupRuleCreate(d, meta)
	}

	return resourceArmSecurityGroupRuleCreate(d, meta)
}

// resourceAzureSecurityGroupRuleReadChoiceFunc goes ahead and chooses which implementation of
// the Read function should be used for reading the Azure security group rule.
func resourceAzureSecurityGroupRuleReadChoiceFunc(d *schema.ResourceData, meta interface{}) error {
	if d.Get("use_asm_api").(bool) {
		return resourceAsmSecurityGroupRuleRead(d, meta)
	}

	return resourceArmSecurityGroupRuleRead(d, meta)
}

// resourceAzureSecurityGroupRuleUpdateChoiceFunc goes ahead and chooses which implementation of
// the Update function should be used for updating the Azure security group rule.
func resourceAzureSecurityGroupRuleUpdateChoiceFunc(d *schema.ResourceData, meta interface{}) error {
	if d.Get("use_asm_api").(bool) {
		return resourceAsmSecurityGroupRuleUpdate(d, meta)
	}

	return resourceArmSecurityGroupRuleUpdate(d, meta)
}

// resourceAzureSecurityGroupRuleDeleteChoiceFunc goes ahead and chooses which implementation of
// the Delete function should be used for deleting the Azure security group rule.
func resourceAzureSecurityGroupRuleDeleteChoiceFunc(d *schema.ResourceData, meta interface{}) error {
	if d.Get("use_asm_api").(bool) {
		return resourceAsmSecurityGroupRuleDelete(d, meta)
	}

	return resourceArmSecurityGroupRuleDelete(d, meta)
}

// resourceAzureStorageServiceCreateChoiceFunc goes ahead and chooses which implementation of
// the Create function should be used for creating the Azure storage service.
func resourceAzureStorageServiceCreateChoiceFunc(d *schema.ResourceData, meta interface{}) error {
	if d.Get("use_asm_api").(bool) {
		return resourceAsmStorageServiceCreate(d, meta)
	}

	return resourceArmStorageServiceCreate(d, meta)
}

// resourceAzureStorageServiceReadChoiceFunc goes ahead and chooses which implementation of
// the Read function should be used for reading the Azure storage service.
func resourceAzureStorageServiceReadChoiceFunc(d *schema.ResourceData, meta interface{}) error {
	if d.Get("use_asm_api").(bool) {
		return resourceAsmStorageServiceRead(d, meta)
	}

	return resourceArmStorageServiceRead(d, meta)
}

// resourceAzureStorageServiceExistsChoiceFunc goes ahead and chooses which implementation of
// the Exists function should be used for checking for the existence of the Azure storage service.
func resourceAzureStorageServiceExistsChoiceFunc(d *schema.ResourceData, meta interface{}) (bool, error) {
	if d.Get("use_asm_api").(bool) {
		return resourceAsmStorageServiceExists(d, meta)
	}

	return resourceArmStorageServiceExists(d, meta)
}

// resourceAzureStorageServiceDeleteChoiceFunc goes ahead and chooses which implementation of
// the Delete function should be used for deleting the Azure storage service.
func resourceAzureStorageServiceDeleteChoiceFunc(d *schema.ResourceData, meta interface{}) error {
	if d.Get("use_asm_api").(bool) {
		return resourceAsmStorageServiceDelete(d, meta)
	}

	return resourceArmStorageServiceDelete(d, meta)
}

// resourceAzureVirtualNetworkCreateChoiceFunc goes ahead and chooses which implementation of
// the Create function should be used for creating the Azure virtual network.
func resourceAzureVirtualNetworkCreateChoiceFunc(d *schema.ResourceData, meta interface{}) error {
	if d.Get("use_asm_api").(bool) {
		return resourceAsmVirtualNetworkCreate(d, meta)
	}

	return resourceArmVirtualNetworkCreate(d, meta)
}

// resourceAzureVirtualNetworkReadChoiceFunc goes ahead and chooses which implementation of
// the Read function should be used for reading the Azure virtual network.
func resourceAzureVirtualNetworkReadChoiceFunc(d *schema.ResourceData, meta interface{}) error {
	if d.Get("use_asm_api").(bool) {
		return resourceAsmVirtualNetworkRead(d, meta)
	}

	return resourceArmVirtualNetworkRead(d, meta)
}

// resourceAzureVirtualNetworkUpdateChoiceFunc goes ahead and chooses which implementation of
// the Update function should be used for updating the Azure virtual network.
func resourceAzureVirtualNetworkUpdateChoiceFunc(d *schema.ResourceData, meta interface{}) error {
	if d.Get("use_asm_api").(bool) {
		return resourceAsmVirtualNetworkUpdate(d, meta)
	}

	return resourceArmVirtualNetworkUpdate(d, meta)
}

// resourceAzureVirtualNetworkDeleteChoiceFunc goes ahead and chooses which implementation of
// the Delete function should be used for deleting the Azure virtual network.
func resourceAzureVirtualNetworkDeleteChoiceFunc(d *schema.ResourceData, meta interface{}) error {
	if d.Get("use_asm_api").(bool) {
		return resourceAsmVirtualNetworkDelete(d, meta)
	}

	return resourceArmVirtualNetworkDelete(d, meta)
}
