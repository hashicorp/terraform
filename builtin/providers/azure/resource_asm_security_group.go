package azure

import (
	"fmt"
	"log"

	"github.com/Azure/azure-sdk-for-go/management"
	"github.com/hashicorp/terraform/helper/schema"
)

func resourceAsmSecurityGroupCreate(d *schema.ResourceData, meta interface{}) (err error) {
	azureClient := meta.(*AzureClient)
	mc := azureClient.asmClient.mgmtClient
	secGroupClient := azureClient.asmClient.secGroupClient

	name := d.Get("name").(string)

	// Compute/set the label
	label := d.Get("label").(string)
	if label == "" {
		label = name
	}

	req, err := secGroupClient.CreateNetworkSecurityGroup(
		name,
		label,
		d.Get("location").(string),
	)
	if err != nil {
		return fmt.Errorf("Error creating Network Security Group %s: %s", name, err)
	}

	if err := mc.WaitForOperation(req, nil); err != nil {
		return fmt.Errorf(
			"Error waiting for Network Security Group %s to be created: %s", name, err)
	}

	d.SetId(name)

	return resourceAsmSecurityGroupRead(d, meta)
}

func resourceAsmSecurityGroupRead(d *schema.ResourceData, meta interface{}) error {
	secGroupClient := meta.(*AzureClient).asmClient.secGroupClient

	sg, err := secGroupClient.GetNetworkSecurityGroup(d.Id())
	if err != nil {
		if management.IsResourceNotFoundError(err) {
			d.SetId("")
			return nil
		}
		return fmt.Errorf("Error retrieving Network Security Group %s: %s", d.Id(), err)
	}

	d.Set("label", sg.Label)
	d.Set("location", sg.Location)

	return nil
}

func resourceAsmSecurityGroupDelete(d *schema.ResourceData, meta interface{}) error {
	azureClient := meta.(*AzureClient)
	mc := azureClient.asmClient.mgmtClient
	secGroupClient := azureClient.asmClient.secGroupClient

	log.Printf("[DEBUG] Deleting Network Security Group: %s", d.Id())
	req, err := secGroupClient.DeleteNetworkSecurityGroup(d.Id())
	if err != nil {
		return fmt.Errorf("Error deleting Network Security Group %s: %s", d.Id(), err)
	}

	// Wait until the network security group is deleted
	if err := mc.WaitForOperation(req, nil); err != nil {
		return fmt.Errorf(
			"Error waiting for Network Security Group %s to be deleted: %s", d.Id(), err)
	}

	d.SetId("")

	return nil
}
