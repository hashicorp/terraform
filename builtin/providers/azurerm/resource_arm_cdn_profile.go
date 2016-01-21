package azurerm

import (
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/Azure/azure-sdk-for-go/arm/cdn"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/helper/schema"
)

func resourceArmCdnProfile() *schema.Resource {
	return &schema.Resource{
		Create: resourceArmCdnProfileCreate,
		Read:   resourceArmCdnProfileRead,
		Update: resourceArmCdnProfileUpdate,
		Delete: resourceArmCdnProfileDelete,

		Schema: map[string]*schema.Schema{
			"name": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"location": &schema.Schema{
				Type:      schema.TypeString,
				Required:  true,
				ForceNew:  true,
				StateFunc: azureRMNormalizeLocation,
			},

			"resource_group_name": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"sku": &schema.Schema{
				Type:         schema.TypeString,
				Required:     true,
				ForceNew:     true,
				ValidateFunc: validateCdnProfileSku,
			},

			"tags": tagsSchema(),
		},
	}
}

func resourceArmCdnProfileCreate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*ArmClient)
	cdnProfilesClient := client.cdnProfilesClient

	log.Printf("[INFO] preparing arguments for Azure ARM CDN Profile creation.")

	name := d.Get("name").(string)
	location := d.Get("location").(string)
	resGroup := d.Get("resource_group_name").(string)
	sku := d.Get("sku").(string)
	tags := d.Get("tags").(map[string]interface{})

	properties := cdn.ProfilePropertiesCreateParameters{
		Sku: &cdn.Sku{
			Name: cdn.SkuName(sku),
		},
	}

	cdnProfile := cdn.ProfileCreateParameters{
		Location:   &location,
		Properties: &properties,
		Tags:       expandTags(tags),
	}

	resp, err := cdnProfilesClient.Create(name, cdnProfile, resGroup)
	if err != nil {
		return err
	}

	d.SetId(*resp.ID)

	log.Printf("[DEBUG] Waiting for CDN Profile (%s) to become available", name)
	stateConf := &resource.StateChangeConf{
		Pending: []string{"Accepted", "Updating", "Creating"},
		Target:  []string{"Succeeded"},
		Refresh: cdnProfileStateRefreshFunc(client, resGroup, name),
		Timeout: 10 * time.Minute,
	}
	if _, err := stateConf.WaitForState(); err != nil {
		return fmt.Errorf("Error waiting for CDN Profile (%s) to become available: %s", name, err)
	}

	return resourceArmCdnProfileRead(d, meta)
}

func resourceArmCdnProfileRead(d *schema.ResourceData, meta interface{}) error {
	cdnProfilesClient := meta.(*ArmClient).cdnProfilesClient

	id, err := parseAzureResourceID(d.Id())
	if err != nil {
		return err
	}
	resGroup := id.ResourceGroup
	name := id.Path["Profiles"]

	resp, err := cdnProfilesClient.Get(name, resGroup)
	if resp.StatusCode == http.StatusNotFound {
		d.SetId("")
		return nil
	}
	if err != nil {
		return fmt.Errorf("Error making Read request on Azure CDN Profile %s: %s", name, err)
	}

	if resp.Properties != nil && resp.Properties.Sku != nil {
		d.Set("sku", string(resp.Properties.Sku.Name))
	}

	flattenAndSetTags(d, resp.Tags)

	return nil
}

func resourceArmCdnProfileUpdate(d *schema.ResourceData, meta interface{}) error {
	cdnProfilesClient := meta.(*ArmClient).cdnProfilesClient

	if !d.HasChange("tags") {
		return nil
	}

	name := d.Get("name").(string)
	resGroup := d.Get("resource_group_name").(string)
	newTags := d.Get("tags").(map[string]interface{})

	props := cdn.ProfileUpdateParameters{
		Tags: expandTags(newTags),
	}

	_, err := cdnProfilesClient.Update(name, props, resGroup)
	if err != nil {
		return fmt.Errorf("Error issuing Azure ARM update request to update CDN Profile %q: %s", name, err)
	}

	return resourceArmCdnProfileRead(d, meta)
}

func resourceArmCdnProfileDelete(d *schema.ResourceData, meta interface{}) error {
	cdnProfilesClient := meta.(*ArmClient).cdnProfilesClient

	id, err := parseAzureResourceID(d.Id())
	if err != nil {
		return err
	}
	resGroup := id.ResourceGroup
	name := id.Path["Profiles"]

	_, err = cdnProfilesClient.DeleteIfExists(name, resGroup)

	return err
}

func cdnProfileStateRefreshFunc(client *ArmClient, resourceGroupName string, cdnProfileName string) resource.StateRefreshFunc {
	return func() (interface{}, string, error) {
		res, err := client.cdnProfilesClient.Get(cdnProfileName, resourceGroupName)
		if err != nil {
			return nil, "", fmt.Errorf("Error issuing read request in cdnProfileStateRefreshFunc to Azure ARM for CND Profile '%s' (RG: '%s'): %s", cdnProfileName, resourceGroupName, err)
		}
		return res, string(res.Properties.ProvisioningState), nil
	}
}

func validateCdnProfileSku(v interface{}, k string) (ws []string, errors []error) {
	value := strings.ToLower(v.(string))
	skus := map[string]bool{
		"standard": true,
		"premium":  true,
	}

	if !skus[value] {
		errors = append(errors, fmt.Errorf("CDN Profile SKU can only be Standard or Premium"))
	}
	return
}
