package azure

import (
	"github.com/Azure/azure-sdk-for-go/arm/resources"
	"github.com/hashicorp/terraform/helper/schema"
)

// resourceArmResourceGroup returns the *schema.Resource
// associated to resource group resources on ARM.
func resourceArmResourceGroup() *schema.Resource {
	return &schema.Resource{
		Create: resourceArmResourceGroupCreate,
		Read:   resourceArmResourceGroupRead,
		Exists: resourceArmResourceGroupExists,
		Delete: resourceArmResourceGroupDelete,

		Schema: map[string]*schema.Schema{
			"name": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
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

// resourceArmResourceGroupCreate goes ahead and creates the specified ARM resource group.
func resourceArmResourceGroupCreate(d *schema.ResourceData, meta interface{}) error {
	resGroupClient := meta.(*AzureClient).armClient.resourceGroupClient

	name := d.Get("name").(string)
	location := d.Get("location").(string)

	_, err := resGroupClient.CreateOrUpdate(
		name,
		resources.ResourceGroup{
			Name:     &name,
			Location: &location,
		},
	)

	return err
}

// resourceArmResourceGroupRead goes ahead and reads the state of the corresponding ARM resource group.
func resourceArmResourceGroupRead(d *schema.ResourceData, meta interface{}) error {
	resGroupClient := meta.(*AzureClient).armClient.resourceGroupClient

	name := d.Get("name").(string)

	res, err := resGroupClient.Get(name)
	if err != nil {
		return err
	}

	// only real thing to check for is location:
	d.Set("location", *res.Location)

	return nil
}

// resourceArmResourceGroupExists goes ahead and checks for the existence of the correspoding ARM resource group.
func resourceArmResourceGroupExists(d *schema.ResourceData, meta interface{}) (bool, error) {
	resGroupClient := meta.(*AzureClient).armClient.resourceGroupClient

	name := d.Get("name").(string)

	resp, err := resGroupClient.CheckExistence(name)
	if err != nil {
		// TODO(aznashwan): implement some error switching helpers in the SDK
		// to avoid HTTP error checks such as the below:
		if resp.StatusCode != 200 {
			return false, err
		} else {
			return true, nil
		}
	}

	return true, nil
}

// resourceArmResourceGroupDelete deletes the specified ARM resource group.
func resourceArmResourceGroupDelete(d *schema.ResourceData, meta interface{}) error {
	resGroupClient := meta.(*AzureClient).armClient.resourceGroupClient

	name := d.Get("name").(string)

	_, err := resGroupClient.Delete(name)
	if err != nil {
		return err
	}

	return nil
}
