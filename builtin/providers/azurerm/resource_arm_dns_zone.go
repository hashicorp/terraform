package azurerm

import (
	"fmt"
	"log"
	"net/http"

	"github.com/Azure/azure-sdk-for-go/arm/dns"
	"github.com/hashicorp/terraform/helper/schema"
)

func resourceArmDnsZone() *schema.Resource {
	return &schema.Resource{
		Create: resourceArmDnsZoneCreate,
		Read:   resourceArmDnsZoneRead,
		Update: resourceArmDnsZoneCreate,
		Delete: resourceArmDnsZoneDelete,

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

			"number_of_record_sets": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
			},

			"max_number_of_record_sets": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
			},
		},
	}
}

func resourceArmDnsZoneCreate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*ArmClient)
	dnsZonesClient := client.dnsZonesClient

	log.Printf("[INFO] preparing arguments for Azure ARM DNS Zone creation.")

	name := d.Get("name").(string)
	location := "global"
	resGroup := d.Get("resource_group_name").(string)

	zone := dns.Zone{
		Location:   &location,
		Properties: &dns.ZoneProperties{},
	}

	zoneParams := dns.ZoneCreateOrUpdateParameters{
		Zone: &zone,
	}

	resp, err := dnsZonesClient.CreateOrUpdate(resGroup, name, zoneParams, "", "*")
	if err != nil {
		return err
	}

	d.SetId(*resp.ID)

	return resourceArmDnsZoneRead(d, meta)
}

func resourceArmDnsZoneRead(d *schema.ResourceData, meta interface{}) error {
	dnsZonesClient := meta.(*ArmClient).dnsZonesClient

	id, err := parseAzureResourceID(d.Id())
	if err != nil {
		return err
	}
	resGroup := id.ResourceGroup
	name := id.Path["dnsZones"]

	resp, err := dnsZonesClient.Get(resGroup, name)
	if resp.StatusCode == http.StatusNotFound {
		d.SetId("")
		return nil
	}
	if err != nil {
		return fmt.Errorf("Error making Read request on Azure DNS Zone %s: %s", name, err)
	}

	if *resp.Properties.MaxNumberOfRecordSets != 0 {
		d.Set("max_number_of_record_sets", *resp.Properties.MaxNumberOfRecordSets)
	}

	if *resp.Properties.NumberOfRecordSets != 0 {
		d.Set("number_of_record_sets", *resp.Properties.NumberOfRecordSets)
	}

	return nil
}

func resourceArmDnsZoneDelete(d *schema.ResourceData, meta interface{}) error {
	dnsZonesClient := meta.(*ArmClient).dnsZonesClient

	id, err := parseAzureResourceID(d.Id())
	if err != nil {
		return err
	}
	resGroup := id.ResourceGroup
	name := id.Path["dnsZones"]

	_, err = dnsZonesClient.Delete(resGroup, name, "")

	return err
}
