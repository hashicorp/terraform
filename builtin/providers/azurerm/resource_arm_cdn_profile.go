package azurerm

import (
	"fmt"
	"log"
	"net/http"

	"strings"

	"github.com/Azure/azure-sdk-for-go/arm/cdn"
	"github.com/hashicorp/terraform/helper/schema"
)

func resourceArmCdnProfile() *schema.Resource {
	return &schema.Resource{
		Create: resourceArmCdnProfileCreate,
		Read:   resourceArmCdnProfileRead,
		Update: resourceArmCdnProfileUpdate,
		Delete: resourceArmCdnProfileDelete,
		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},

		Schema: map[string]*schema.Schema{
			"name": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"location": {
				Type:      schema.TypeString,
				Required:  true,
				ForceNew:  true,
				StateFunc: azureRMNormalizeLocation,
			},

			"resource_group_name": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"sku": {
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

	cdnProfile := cdn.ProfileCreateParameters{
		Location: &location,
		Tags:     expandTags(tags),
		Sku: &cdn.Sku{
			Name: cdn.SkuName(sku),
		},
	}

	_, err := cdnProfilesClient.Create(name, cdnProfile, resGroup, make(chan struct{}))
	if err != nil {
		return err
	}

	read, err := cdnProfilesClient.Get(name, resGroup)
	if err != nil {
		return err
	}
	if read.ID == nil {
		return fmt.Errorf("Cannot read CND Profile %s (resource group %s) ID", name, resGroup)
	}

	d.SetId(*read.ID)

	return resourceArmCdnProfileRead(d, meta)
}

func resourceArmCdnProfileRead(d *schema.ResourceData, meta interface{}) error {
	cdnProfilesClient := meta.(*ArmClient).cdnProfilesClient

	id, err := parseAzureResourceID(d.Id())
	if err != nil {
		return err
	}
	resGroup := id.ResourceGroup
	name := id.Path["profiles"]

	resp, err := cdnProfilesClient.Get(name, resGroup)
	if err != nil {
		if resp.StatusCode == http.StatusNotFound {
			d.SetId("")
			return nil
		}
		return fmt.Errorf("Error making Read request on Azure CDN Profile %s: %s", name, err)
	}

	d.Set("name", name)
	d.Set("resource_group_name", resGroup)
	d.Set("location", azureRMNormalizeLocation(*resp.Location))

	if resp.Sku != nil {
		d.Set("sku", string(resp.Sku.Name))
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

	_, err := cdnProfilesClient.Update(name, props, resGroup, make(chan struct{}))
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
	name := id.Path["profiles"]

	_, err = cdnProfilesClient.DeleteIfExists(name, resGroup, make(chan struct{}))

	return err
}

func validateCdnProfileSku(v interface{}, k string) (ws []string, errors []error) {
	value := strings.ToLower(v.(string))
	skus := map[string]bool{
		"standard_akamai":  true,
		"premium_verizon":  true,
		"standard_verizon": true,
	}

	if !skus[value] {
		errors = append(errors, fmt.Errorf("CDN Profile SKU can only be Premium_Verizon, Standard_Verizon or Standard_Akamai"))
	}
	return
}
