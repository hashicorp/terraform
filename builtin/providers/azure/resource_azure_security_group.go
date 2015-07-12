package azure

import (
	"fmt"
	"log"

	"github.com/Azure/azure-sdk-for-go/management"
	"github.com/hashicorp/terraform/helper/schema"
)

func resourceAzureSecurityGroup() *schema.Resource {
	return &schema.Resource{
		Create: resourceAzureSecurityGroupCreate,
		Read:   resourceAzureSecurityGroupRead,
		Delete: resourceAzureSecurityGroupDelete,

		Schema: map[string]*schema.Schema{
			"name": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"label": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
				ForceNew: true,
			},

			"location": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
		},
	}
}

func resourceAzureSecurityGroupCreate(d *schema.ResourceData, meta interface{}) (err error) {
	azureClient := meta.(*Client)
	mc := azureClient.mgmtClient
	secGroupClient := azureClient.secGroupClient

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

	return resourceAzureSecurityGroupRead(d, meta)
}

func resourceAzureSecurityGroupRead(d *schema.ResourceData, meta interface{}) error {
	secGroupClient := meta.(*Client).secGroupClient

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

func resourceAzureSecurityGroupDelete(d *schema.ResourceData, meta interface{}) error {
	azureClient := meta.(*Client)
	mc := azureClient.mgmtClient
	secGroupClient := azureClient.secGroupClient

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
