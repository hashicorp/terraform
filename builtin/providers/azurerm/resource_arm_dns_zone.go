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
		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},

		Schema: map[string]*schema.Schema{
			"name": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"resource_group_name": &schema.Schema{
				Type:             schema.TypeString,
				Required:         true,
				ForceNew:         true,
				DiffSuppressFunc: resourceAzurermResourceGroupNameDiffSuppress,
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

			"name_servers": &schema.Schema{
				Type:     schema.TypeSet,
				Computed: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
				Set:      schema.HashString,
			},

			"tags": tagsSchema(),
		},
	}
}

func resourceArmDnsZoneCreate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*ArmClient)
	rivieraClient := client.rivieraClient

	tags := d.Get("tags").(map[string]interface{})
	expandedTags := expandTags(tags)

	createRequest := rivieraClient.NewRequest()
	createRequest.Command = &dns.CreateDNSZone{
		Name:              d.Get("name").(string),
		Location:          "global",
		ResourceGroupName: d.Get("resource_group_name").(string),
		Tags:              *expandedTags,
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

	id, err := parseAzureResourceID(d.Id())
	if err != nil {
		return err
	}
	resGroup := id.ResourceGroup

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

	d.Set("resource_group_name", resGroup)
	d.Set("number_of_record_sets", resp.NumberOfRecordSets)
	d.Set("max_number_of_record_sets", resp.MaxNumberOfRecordSets)
	d.Set("name", resp.Name)

	nameServers := make([]string, 0, len(resp.NameServers))
	for _, ns := range resp.NameServers {
		nameServers = append(nameServers, *ns)
	}
	if err := d.Set("name_servers", nameServers); err != nil {
		return err
	}

	flattenAndSetTags(d, resp.Tags)

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
