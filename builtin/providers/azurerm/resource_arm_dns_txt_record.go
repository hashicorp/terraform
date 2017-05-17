package azurerm

import (
	"fmt"
	"log"

	"github.com/hashicorp/terraform/helper/schema"
	"github.com/jen20/riviera/dns"
)

func resourceArmDnsTxtRecord() *schema.Resource {
	return &schema.Resource{
		Create: resourceArmDnsTxtRecordCreate,
		Read:   resourceArmDnsTxtRecordRead,
		Update: resourceArmDnsTxtRecordCreate,
		Delete: resourceArmDnsTxtRecordDelete,
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
						"value": &schema.Schema{
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

func resourceArmDnsTxtRecordCreate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*ArmClient)
	rivieraClient := client.rivieraClient

	tags := d.Get("tags").(map[string]interface{})
	expandedTags := expandTags(tags)

	createCommand := &dns.CreateTXTRecordSet{
		Name:              d.Get("name").(string),
		Location:          "global",
		ResourceGroupName: d.Get("resource_group_name").(string),
		ZoneName:          d.Get("zone_name").(string),
		TTL:               d.Get("ttl").(int),
		Tags:              *expandedTags,
	}

	txtRecords, recordErr := expandAzureRmDnsTxtRecords(d)
	if recordErr != nil {
		return fmt.Errorf("Error Building list of Azure RM Txt Records: %s", recordErr)
	}
	createCommand.TXTRecords = txtRecords

	createRequest := rivieraClient.NewRequest()
	createRequest.Command = createCommand

	createResponse, err := createRequest.Execute()
	if err != nil {
		return fmt.Errorf("Error creating DNS TXT Record: %s", err)
	}
	if !createResponse.IsSuccessful() {
		return fmt.Errorf("Error creating DNS TXT Record: %s", createResponse.Error)
	}

	readRequest := rivieraClient.NewRequest()
	readRequest.Command = &dns.GetTXTRecordSet{
		Name:              d.Get("name").(string),
		ResourceGroupName: d.Get("resource_group_name").(string),
		ZoneName:          d.Get("zone_name").(string),
	}

	readResponse, err := readRequest.Execute()
	if err != nil {
		return fmt.Errorf("Error reading DNS TXT Record: %s", err)
	}
	if !readResponse.IsSuccessful() {
		return fmt.Errorf("Error reading DNS TXT Record: %s", readResponse.Error)
	}

	resp := readResponse.Parsed.(*dns.GetTXTRecordSetResponse)
	d.SetId(resp.ID)

	return resourceArmDnsTxtRecordRead(d, meta)
}

func resourceArmDnsTxtRecordRead(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*ArmClient)
	rivieraClient := client.rivieraClient

	id, err := parseAzureResourceID(d.Id())
	if err != nil {
		return err
	}

	readRequest := rivieraClient.NewRequestForURI(d.Id())
	readRequest.Command = &dns.GetTXTRecordSet{}

	readResponse, err := readRequest.Execute()
	if err != nil {
		return fmt.Errorf("Error reading DNS TXT Record: %s", err)
	}
	if !readResponse.IsSuccessful() {
		log.Printf("[INFO] Error reading DNS TXT Record %q - removing from state", d.Id())
		d.SetId("")
		return fmt.Errorf("Error reading DNS TXT Record: %s", readResponse.Error)
	}

	resp := readResponse.Parsed.(*dns.GetTXTRecordSetResponse)

	d.Set("name", resp.Name)
	d.Set("resource_group_name", id.ResourceGroup)
	d.Set("zone_name", id.Path["dnszones"])
	d.Set("ttl", resp.TTL)

	if resp.TXTRecords != nil {
		if err := d.Set("record", flattenAzureRmDnsTxtRecords(resp.TXTRecords)); err != nil {
			log.Printf("[INFO] Error setting the Azure RM TXT Record State: %s", err)
			return err
		}
	}

	flattenAndSetTags(d, &resp.Tags)

	return nil
}

func resourceArmDnsTxtRecordDelete(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*ArmClient)
	rivieraClient := client.rivieraClient

	deleteRequest := rivieraClient.NewRequestForURI(d.Id())
	deleteRequest.Command = &dns.DeleteRecordSet{
		RecordSetType: "TXT",
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

func expandAzureRmDnsTxtRecords(d *schema.ResourceData) ([]dns.TXTRecord, error) {
	configs := d.Get("record").(*schema.Set).List()
	txtRecords := make([]dns.TXTRecord, 0, len(configs))

	for _, configRaw := range configs {
		data := configRaw.(map[string]interface{})

		txtRecord := dns.TXTRecord{
			Value: data["value"].(string),
		}

		txtRecords = append(txtRecords, txtRecord)

	}

	return txtRecords, nil

}

func flattenAzureRmDnsTxtRecords(records []dns.TXTRecord) []map[string]interface{} {
	result := make([]map[string]interface{}, 0, len(records))
	for _, record := range records {
		txtRecord := make(map[string]interface{})
		txtRecord["value"] = record.Value

		result = append(result, txtRecord)
	}
	return result
}
