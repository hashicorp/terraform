package azurerm

import (
	"fmt"
	"log"
	"net/http"

	"github.com/Azure/azure-sdk-for-go/arm/web"
	"github.com/hashicorp/terraform/helper/schema"
)

func resourceArmAppServicePlan() *schema.Resource {
	return &schema.Resource{
		Create: resourceArmAppServicePlanCreateUpdate,
		Read:   resourceArmAppServicePlanRead,
		Update: resourceArmAppServicePlanCreateUpdate,
		Delete: resourceArmAppServicePlanDelete,

		Schema: map[string]*schema.Schema{
			"resource_group_name": {
				Type:     schema.TypeString,
				Required: true,
			},
			"name": {
				Type:     schema.TypeString,
				Required: true,
			},
			"location": {
				Type:     schema.TypeString,
				Required: true,
			},
			"tier": {
				Type:     schema.TypeString,
				Required: true,
			},
			"maximum_number_of_workers": {
				Type:     schema.TypeString,
				Optional: true,
			},
		},
	}
}

func resourceArmAppServicePlanCreateUpdate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*ArmClient)
	AppServicePlanClient := client.appServicePlansClient

	log.Printf("[INFO] preparing arguments for Azure ARM Server Farm creation.")

	resGroup := d.Get("resource_group_name").(string)
	name := d.Get("name").(string)
	location := d.Get("location").(string)
	tier := d.Get("tier").(string)

	sku := web.SkuDescription{
		Name: &tier,
	}

	properties := web.AppServicePlanProperties{}
	if v, ok := d.GetOk("maximum_number_of_workers"); ok {
		maximumNumberOfWorkers := v.(int32)
		properties.MaximumNumberOfWorkers = &maximumNumberOfWorkers
	}

	appServicePlan := web.AppServicePlan{
		Location:                 &location,
		AppServicePlanProperties: &properties,
		Sku: &sku,
	}

	_, err := AppServicePlanClient.CreateOrUpdate(resGroup, name, appServicePlan, make(chan struct{}))
	if err != nil {
		return err
	}

	read, err := AppServicePlanClient.Get(resGroup, name)
	if err != nil {
		return err
	}
	if read.ID == nil {
		return fmt.Errorf("Cannot read Server farm %s (resource group %s) ID", name, resGroup)
	}

	d.SetId(*read.ID)

	return resourceArmAppServicePlanRead(d, meta)
}

func resourceArmAppServicePlanRead(d *schema.ResourceData, meta interface{}) error {
	AppServicePlanClient := meta.(*ArmClient).appServicePlansClient

	id, err := parseAzureResourceID(d.Id())
	if err != nil {
		return err
	}

	log.Printf("[DEBUG] Reading app service plan %s", id)

	resGroup := id.ResourceGroup
	name := id.Path["serverfarms"]

	resp, err := AppServicePlanClient.Get(resGroup, name)
	if err != nil {
		if resp.StatusCode == http.StatusNotFound {
			d.SetId("")
			return nil
		}
		return fmt.Errorf("Error making Read request on Azure Server Farm %s: %s", name, err)
	}

	d.Set("name", name)
	d.Set("resource_group_name", resGroup)

	return nil
}

func resourceArmAppServicePlanDelete(d *schema.ResourceData, meta interface{}) error {
	AppServicePlanClient := meta.(*ArmClient).appServicePlansClient

	id, err := parseAzureResourceID(d.Id())
	if err != nil {
		return err
	}
	resGroup := id.ResourceGroup
	name := id.Path["serverfarms"]

	log.Printf("[DEBUG] Deleting app service plan %s: %s", resGroup, name)

	_, err = AppServicePlanClient.Delete(resGroup, name)

	return err
}
