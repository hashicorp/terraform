package azurerm

import (
	"fmt"
	"log"
	"net/http"
	"strings"

	"github.com/Azure/azure-sdk-for-go/arm/compute"
	"github.com/hashicorp/terraform/helper/schema"
	"github.com/jen20/riviera/azure"
)

func resourceArmAvailabilitySet() *schema.Resource {
	return &schema.Resource{
		Create: resourceArmAvailabilitySetCreate,
		Read:   resourceArmAvailabilitySetRead,
		Update: resourceArmAvailabilitySetCreate,
		Delete: resourceArmAvailabilitySetDelete,
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

			"location": locationSchema(),

			"platform_update_domain_count": {
				Type:     schema.TypeInt,
				Optional: true,
				Default:  5,
				ForceNew: true,
				ValidateFunc: func(v interface{}, k string) (ws []string, errors []error) {
					value := v.(int)
					if value > 20 {
						errors = append(errors, fmt.Errorf(
							"Maximum value for `platform_update_domain_count` is 20"))
					}
					return
				},
			},

			"platform_fault_domain_count": {
				Type:     schema.TypeInt,
				Optional: true,
				Default:  3,
				ForceNew: true,
				ValidateFunc: func(v interface{}, k string) (ws []string, errors []error) {
					value := v.(int)
					if value > 3 {
						errors = append(errors, fmt.Errorf(
							"Maximum value for (%s) is 3", k))
					}
					return
				},
			},

			"managed": {
				Type:     schema.TypeBool,
				Optional: true,
				Default:  false,
				ForceNew: true,
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
	managed := d.Get("managed").(bool)

	availSet := compute.AvailabilitySet{
		Name:     &name,
		Location: &location,
		AvailabilitySetProperties: &compute.AvailabilitySetProperties{
			PlatformFaultDomainCount:  azure.Int32(int32(faultDomainCount)),
			PlatformUpdateDomainCount: azure.Int32(int32(updateDomainCount)),
		},
		Tags: expandTags(tags),
	}

	if managed == true {
		n := "Aligned"
		availSet.Sku = &compute.Sku{
			Name: &n,
		}
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
	if err != nil {
		if resp.StatusCode == http.StatusNotFound {
			d.SetId("")
			return nil
		}
		return fmt.Errorf("Error making Read request on Azure Availability Set %s: %s", name, err)
	}

	availSet := *resp.AvailabilitySetProperties
	d.Set("resource_group_name", resGroup)
	d.Set("platform_update_domain_count", availSet.PlatformUpdateDomainCount)
	d.Set("platform_fault_domain_count", availSet.PlatformFaultDomainCount)
	d.Set("name", resp.Name)
	d.Set("location", resp.Location)

	if resp.Sku != nil && resp.Sku.Name != nil {
		d.Set("managed", strings.EqualFold(*resp.Sku.Name, "Aligned"))
	}

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
