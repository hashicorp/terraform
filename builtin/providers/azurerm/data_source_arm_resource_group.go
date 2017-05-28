package azurerm

import (
	"fmt"

	"github.com/hashicorp/errwrap"
	"github.com/hashicorp/terraform/helper/schema"
)

func dataSourceArmResourceGroup() *schema.Resource {
	return &schema.Resource{
		Read: dataSourceArmResourceGroupRead,

		Schema: map[string]*schema.Schema{
			"name": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},

			"location": locationForDataSourceSchema(),
			"tags":     tagsForDataSourceSchema(),
		},
	}
}

func dataSourceArmResourceGroupRead(d *schema.ResourceData, meta interface{}) error {
	armClient := meta.(*ArmClient)
	resourceGroupClient := armClient.resourceGroupClient

	resourceGroupName := d.Get("name").(string)
	result, err := resourceGroupClient.Get(resourceGroupName)
	if err != nil {
		return errwrap.Wrapf("Error reading Resource Group {{err}}", err)
	}

	if v, ok := d.GetOk("location"); ok {
		location := azureRMNormalizeLocation(v.(string))
		actualLocation := azureRMNormalizeLocation(*result.Location)

		if location != actualLocation {
			return fmt.Errorf(`Resource Group's location "%s" doesn't match the location specified in Data Source: "%s"`,
				actualLocation, location)
		}
	}

	d.Set("location", *result.Location)
	flattenAndSetTags(d, result.Tags)
	d.SetId(*result.ID)

	return nil
}
