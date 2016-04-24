package azurerm

import (
	"fmt"
	"log"

	"github.com/hashicorp/terraform/helper/schema"
	"github.com/jen20/riviera/dns"
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
	rivieraClient := client.rivieraClient

	createRequest := rivieraClient.NewRequest()
	createRequest.Command = &dns.CreateDNSZone{
		Name:              d.Get("name").(string),
		Location:          "global",
		ResourceGroupName: d.Get("resource_group_name").(string),
	}

	createResponse, err := createRequest.Execute()
	if err != nil {
		return fmt.Errorf("Error creating DNS Zone: %s", err)
	}
	if !createResponse.IsSuccessful() {
		return fmt.Errorf("Error creating DNS Zone: %s", createResponse.Error)
	}

	readRequest := rivieraClient.NewRequest()
	readRequest.Command = &dns.GetDNSZone{
		Name:              d.Get("name").(string),
		ResourceGroupName: d.Get("resource_group_name").(string),
	}

	readResponse, err := readRequest.Execute()
	if err != nil {
		return fmt.Errorf("Error reading DNS Zone: %s", err)
	}
	if !readResponse.IsSuccessful() {
		return fmt.Errorf("Error reading DNS Zone: %s", readResponse.Error)
	}

	resp := readResponse.Parsed.(*dns.GetDNSZoneResponse)
	d.SetId(*resp.ID)

	return resourceArmDnsZoneRead(d, meta)
}

func resourceArmDnsZoneRead(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*ArmClient)
	rivieraClient := client.rivieraClient

	readRequest := rivieraClient.NewRequestForURI(d.Id())
	readRequest.Command = &dns.GetDNSZone{}

	readResponse, err := readRequest.Execute()
	if err != nil {
		return fmt.Errorf("Error reading DNS Zone: %s", err)
	}
	if !readResponse.IsSuccessful() {
		log.Printf("[INFO] Error reading DNS Zone %q - removing from state", d.Id())
		d.SetId("")
		return fmt.Errorf("Error reading DNS Zone: %s", readResponse.Error)
	}

	resp := readResponse.Parsed.(*dns.GetDNSZoneResponse)

	d.Set("number_of_record_sets", resp.NumberOfRecordSets)
	d.Set("max_number_of_record_sets", resp.MaxNumberOfRecordSets)

	return nil
}

func resourceArmDnsZoneDelete(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*ArmClient)
	rivieraClient := client.rivieraClient

	deleteRequest := rivieraClient.NewRequestForURI(d.Id())
	deleteRequest.Command = &dns.DeleteDNSZone{}

	deleteResponse, err := deleteRequest.Execute()
	if err != nil {
		return fmt.Errorf("Error deleting DNS Zone: %s", err)
	}
	if !deleteResponse.IsSuccessful() {
		return fmt.Errorf("Error deleting DNS Zone: %s", deleteResponse.Error)
	}

	return nil
}
