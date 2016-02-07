package azurerm

import (
	"fmt"
	"log"

	"github.com/hashicorp/terraform/helper/schema"
	"github.com/jen20/riviera/dns"
)

func resourceArmDnsNsRecord() *schema.Resource {
	return &schema.Resource{
		Create: resourceArmDnsNsRecordCreate,
		Read:   resourceArmDnsNsRecordRead,
		Update: resourceArmDnsNsRecordCreate,
		Delete: resourceArmDnsNsRecordDelete,

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

			"zone_name": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},

			"record": &schema.Schema{
				Type:     schema.TypeSet,
				Required: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"nsdname": &schema.Schema{
							Type:     schema.TypeString,
							Required: true,
						},
					},
				},
			},

			"ttl": &schema.Schema{
				Type:     schema.TypeInt,
				Required: true,
			},

			"tags": tagsSchema(),
		},
	}
}

func resourceArmDnsNsRecordCreate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*ArmClient)
	rivieraClient := client.rivieraClient

	tags := d.Get("tags").(map[string]interface{})
	expandedTags := expandTags(tags)

	createCommand := &dns.CreateNSRecordSet{
		Name:              d.Get("name").(string),
		Location:          "global",
		ResourceGroupName: d.Get("resource_group_name").(string),
		ZoneName:          d.Get("zone_name").(string),
		TTL:               d.Get("ttl").(int),
		Tags:              *expandedTags,
	}

	nsRecords, recordErr := expandAzureRmDnsNsRecords(d)
	if recordErr != nil {
		return fmt.Errorf("Error Building list of Azure RM NS Records: %s", recordErr)
	}
	createCommand.NSRecords = nsRecords

	createRequest := rivieraClient.NewRequest()
	createRequest.Command = createCommand

	createResponse, err := createRequest.Execute()
	if err != nil {
		return fmt.Errorf("Error creating DNS NS Record: %s", err)
	}
	if !createResponse.IsSuccessful() {
		return fmt.Errorf("Error creating DNS NS Record: %s", createResponse.Error)
	}

	readRequest := rivieraClient.NewRequest()
	readRequest.Command = &dns.GetNSRecordSet{
		Name:              d.Get("name").(string),
		ResourceGroupName: d.Get("resource_group_name").(string),
		ZoneName:          d.Get("zone_name").(string),
	}

	readResponse, err := readRequest.Execute()
	if err != nil {
		return fmt.Errorf("Error reading DNS NS Record: %s", err)
	}
	if !readResponse.IsSuccessful() {
		return fmt.Errorf("Error reading DNS NS Record: %s", readResponse.Error)
	}

	resp := readResponse.Parsed.(*dns.GetNSRecordSetResponse)
	d.SetId(resp.ID)

	return resourceArmDnsNsRecordRead(d, meta)
}

func resourceArmDnsNsRecordRead(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*ArmClient)
	rivieraClient := client.rivieraClient

	readRequest := rivieraClient.NewRequestForURI(d.Id())
	readRequest.Command = &dns.GetNSRecordSet{}

	readResponse, err := readRequest.Execute()
	if err != nil {
		return fmt.Errorf("Error reading DNS Ns Record: %s", err)
	}
	if !readResponse.IsSuccessful() {
		log.Printf("[INFO] Error reading DNS NS Record %q - removing from state", d.Id())
		d.SetId("")
		return fmt.Errorf("Error reading DNS NS Record: %s", readResponse.Error)
	}

	resp := readResponse.Parsed.(*dns.GetNSRecordSetResponse)

	d.Set("ttl", resp.TTL)

	if resp.NSRecords != nil {
		if err := d.Set("record", flattenAzureRmDnsNsRecords(resp.NSRecords)); err != nil {
			log.Printf("[INFO] Error setting the Azure RM NS Record State: %s", err)
			return err
		}
	}

	flattenAndSetTags(d, &resp.Tags)

	return nil
}

func resourceArmDnsNsRecordDelete(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*ArmClient)
	rivieraClient := client.rivieraClient

	deleteRequest := rivieraClient.NewRequestForURI(d.Id())
	deleteRequest.Command = &dns.DeleteRecordSet{
		RecordSetType: "NS",
	}

	deleteResponse, err := deleteRequest.Execute()
	if err != nil {
		return fmt.Errorf("Error deleting DNS TXT Record: %s", err)
	}
	if !deleteResponse.IsSuccessful() {
		return fmt.Errorf("Error deleting DNS TXT Record: %s", deleteResponse.Error)
	}

	return nil
}

func expandAzureRmDnsNsRecords(d *schema.ResourceData) ([]dns.NSRecord, error) {
	configs := d.Get("record").(*schema.Set).List()
	nsRecords := make([]dns.NSRecord, 0, len(configs))

	for _, configRaw := range configs {
		data := configRaw.(map[string]interface{})

		nsRecord := dns.NSRecord{
			NSDName: data["nsdname"].(string),
		}

		nsRecords = append(nsRecords, nsRecord)

	}

	return nsRecords, nil

}

func flattenAzureRmDnsNsRecords(records []dns.NSRecord) []map[string]interface{} {
	result := make([]map[string]interface{}, 0, len(records))
	for _, record := range records {
		nsRecord := make(map[string]interface{})
		nsRecord["nsdname"] = record.NSDName

		result = append(result, nsRecord)
	}
	return result
}
