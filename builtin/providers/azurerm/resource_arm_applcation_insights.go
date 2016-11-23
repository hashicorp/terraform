package azurerm

import (
	"fmt"
	"log"

	"net/http"

	"strings"

	"github.com/Azure/azure-sdk-for-go/arm/applicationinsights"
	"github.com/hashicorp/terraform/helper/schema"
)

func resourceArmApplicationInsights() *schema.Resource {
	return &schema.Resource{
		Create: resourceArmApplicationInsightsCreate,
		Read:   resourceArmApplicationInsightsRead,
		Update: resourceArmApplicationInsightsCreate,
		Delete: resourceArmApplicationInsightsDelete,
		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},

		Schema: map[string]*schema.Schema{
			"name": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"resource_group_name": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"location": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"application_id": {
				Type:     schema.TypeString,
				Required: true,
			},

			"application_type": {
				Type:         schema.TypeString,
				Required:     true,
				ValidateFunc: validateApplicationInsightsApplicationType,
			},

			"app_id": {
				Type:     schema.TypeString,
				Computed: true,
			},

			"instrumentation_key": {
				Type:     schema.TypeString,
				Computed: true,
			},
		},
	}
}

func resourceArmApplicationInsightsCreate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*ArmClient).applicationInsightsClient
	log.Printf("[INFO] preparing arguments for Azure ARM Application Insights creation.")

	name := d.Get("name").(string)
	location := d.Get("location").(string)
	resGroup := d.Get("resource_group_name").(string)

	applicationId := d.Get("application_id").(string)
	applicationType := d.Get("application_type").(string)

	parameters := applicationinsights.Resource{
		Location: &location,
		Properties: &applicationinsights.Properties{
			ApplicationID:   &applicationId,
			ApplicationType: applicationinsights.ApplicationType(applicationType),
		},
		Kind: &applicationType, // TODO: make an enum
	}

	_, err := client.CreateOrUpdate(resGroup, name, parameters)
	if err != nil {
		return err
	}

	read, err := client.Get(resGroup, name)
	if err != nil {
		return err
	}

	if read.ID == nil {
		return fmt.Errorf("Cannot read Application Insights instance %s (resource group %s) ID", name, resGroup)
	}

	d.SetId(*read.ID)

	return resourceArmApplicationInsightsRead(d, meta)
}

func resourceArmApplicationInsightsRead(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*ArmClient).applicationInsightsClient

	id, err := parseAzureResourceID(d.Id())
	if err != nil {
		return err
	}
	resGroup := id.ResourceGroup
	name := id.Path["components"]

	resp, err := client.Get(resGroup, name)
	if err != nil {
		return fmt.Errorf("Error making Read request on Azure Application Insights instance %s: %s", name, err)
	}
	if resp.StatusCode == http.StatusNotFound {
		d.SetId("")
		return nil
	}

	d.Set("name", resp.Name)
	d.Set("resource_group_name", resGroup)
	d.Set("location", azureRMNormalizeLocation(*resp.Location))

	d.Set("application_id", resp.Properties.ApplicationID)
	d.Set("application_type", resp.Properties.ApplicationType)

	d.Set("app_id", resp.Properties.AppID)
	d.Set("instrumentation_key", resp.Properties.InstrumentationKey)

	return nil
}

func resourceArmApplicationInsightsDelete(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*ArmClient).applicationInsightsClient

	id, err := parseAzureResourceID(d.Id())
	if err != nil {
		return err
	}
	resGroup := id.ResourceGroup
	name := id.Path["components"]

	resp, err := client.Delete(resGroup, name)

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("Error issuing Azure ARM delete request of Application Insights instance '%s': %s", name, err)
	}

	return nil
}

func validateApplicationInsightsApplicationType(v interface{}, k string) (ws []string, errors []error) {
	value := strings.ToLower(v.(string))
	skus := map[string]bool{
		"web":   true,
		"other": true,
	}

	if !skus[value] {
		errors = append(errors, fmt.Errorf("Application Insights Application Type can only be Web or Other"))
	}
	return
}
