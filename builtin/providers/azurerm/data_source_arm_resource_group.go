package azurerm

import (
	"fmt"

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

	resourceGroupName := d.Get("name").(string)
	location, getLocationOk := d.GetOk("location")
	resourceId := fmt.Sprintf("/subscriptions/%s/resourceGroups/%s", armClient.subscriptionId, resourceGroupName)

	d.SetId(resourceId)

	if err := resourceArmResourceGroupRead(d, meta); err != nil {
		return err
	}

	if getLocationOk {
		actualLocation := azureRMNormalizeLocation(d.Get("location").(string))
		location := azureRMNormalizeLocation(location)

		if location != actualLocation {
			return fmt.Errorf(`The location specified in Data Source (%s) doesn't match the actual location of the Resource Group "%s (%s)"`,
				location, resourceGroupName, actualLocation)
		}
	}

	return nil
}
