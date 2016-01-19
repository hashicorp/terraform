package azurerm

import (
	"fmt"
	"log"
	"net/http"

	"github.com/Azure/azure-sdk-for-go/arm/compute"
	"github.com/hashicorp/terraform/helper/schema"
)

func resourceArmAvailabilitySet() *schema.Resource {
	return &schema.Resource{
		Create: resourceArmAvailabilitySetCreate,
		Read:   resourceArmAvailabilitySetRead,
		Update: resourceArmAvailabilitySetCreate,
		Delete: resourceArmAvailabilitySetDelete,

		Schema: map[string]*schema.Schema{
			"name": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"resource_group_name": &schema.Schema{
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

			"platform_update_domain_count": &schema.Schema{
				Type:     schema.TypeInt,
				Optional: true,
				Default:  5,
				ValidateFunc: func(v interface{}, k string) (ws []string, errors []error) {
					value := v.(int)
					if value > 20 {
						errors = append(errors, fmt.Errorf(
							"Maximum value for `platform_update_domain_count` is 20"))
					}
					return
				},
			},

			"platform_fault_domain_count": &schema.Schema{
				Type:     schema.TypeInt,
				Optional: true,
				Default:  3,
				ValidateFunc: func(v interface{}, k string) (ws []string, errors []error) {
					value := v.(int)
					if value > 3 {
						errors = append(errors, fmt.Errorf(
							"Maximum value for (%s) is 3", k))
					}
					return
				},
			},

			"tags": tagsSchema(),
		},
	}
}

func resourceArmAvailabilitySetCreate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*ArmClient)
	availSetClient := client.availSetClient

	log.Printf("[INFO] preparing arguments for Azure ARM Availability Set creation.")

	name := d.Get("name").(string)
	location := d.Get("location").(string)
	resGroup := d.Get("resource_group_name").(string)
	updateDomainCount := d.Get("platform_update_domain_count").(int)
	faultDomainCount := d.Get("platform_fault_domain_count").(int)
	tags := d.Get("tags").(map[string]interface{})

	availSet := compute.AvailabilitySet{
		Name:     &name,
		Location: &location,
		Properties: &compute.AvailabilitySetProperties{
			PlatformFaultDomainCount:  &faultDomainCount,
			PlatformUpdateDomainCount: &updateDomainCount,
		},
		Tags: expandTags(tags),
	}

	resp, err := availSetClient.CreateOrUpdate(resGroup, name, availSet)
	if err != nil {
		return err
	}

	d.SetId(*resp.ID)

	return resourceArmAvailabilitySetRead(d, meta)
}

func resourceArmAvailabilitySetRead(d *schema.ResourceData, meta interface{}) error {
	availSetClient := meta.(*ArmClient).availSetClient

	id, err := parseAzureResourceID(d.Id())
	if err != nil {
		return err
	}
	resGroup := id.ResourceGroup
	name := id.Path["availabilitySets"]

	resp, err := availSetClient.Get(resGroup, name)
	if resp.StatusCode == http.StatusNotFound {
		d.SetId("")
		return nil
	}
	if err != nil {
		return fmt.Errorf("Error making Read request on Azure Availability Set %s: %s", name, err)
	}

	availSet := *resp.Properties
	d.Set("platform_update_domain_count", availSet.PlatformUpdateDomainCount)
	d.Set("platform_fault_domain_count", availSet.PlatformFaultDomainCount)

	flattenAndSetTags(d, resp.Tags)

	return nil
}

func resourceArmAvailabilitySetDelete(d *schema.ResourceData, meta interface{}) error {
	availSetClient := meta.(*ArmClient).availSetClient

	id, err := parseAzureResourceID(d.Id())
	if err != nil {
		return err
	}
	resGroup := id.ResourceGroup
	name := id.Path["availabilitySets"]

	_, err = availSetClient.Delete(resGroup, name)

	return err
}
