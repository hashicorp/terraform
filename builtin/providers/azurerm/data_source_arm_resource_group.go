package azurerm

import (
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
	resourceId := &ResourceID{
		SubscriptionID: armClient.subscriptionId,
		ResourceGroup:  resourceGroupName,
	}
	resourceIdString, err := composeAzureResourceID(resourceId)

	if err != nil {
		return err
	}

	d.SetId(resourceIdString)

	if err := resourceArmResourceGroupRead(d, meta); err != nil {
		return err
	}

	return nil
}
