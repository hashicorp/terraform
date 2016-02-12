package azurerm

import (
	"fmt"
	"log"

	"github.com/hashicorp/terraform/helper/schema"
	"github.com/jen20/riviera/azure"
	"github.com/jen20/riviera/search"
)

func resourceArmSearchService() *schema.Resource {
	return &schema.Resource{
		Create: resourceArmSearchServiceCreate,
		Read:   resourceArmSearchServiceRead,
		Update: resourceArmSearchServiceCreate,
		Delete: resourceArmSearchServiceDelete,

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
				Type:     schema.TypeString,
				Required: true,
			},

			"replica_count": &schema.Schema{
				Type:     schema.TypeInt,
				Optional: true,
				Computed: true,
			},

			"partition_count": &schema.Schema{
				Type:     schema.TypeInt,
				Optional: true,
				Computed: true,
			},

			"tags": tagsSchema(),
		},
	}
}

func resourceArmSearchServiceCreate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*ArmClient)
	rivieraClient := client.rivieraClient

	tags := d.Get("tags").(map[string]interface{})
	expandedTags := expandTags(tags)

	command := &search.CreateOrUpdateSearchService{
		Name:              d.Get("name").(string),
		Location:          d.Get("location").(string),
		ResourceGroupName: d.Get("resource_group_name").(string),
		Tags:              *expandedTags,
		Sku: search.Sku{
			Name: d.Get("sku").(string),
		},
	}

	if v, ok := d.GetOk("replica_count"); ok {
		command.ReplicaCount = azure.String(v.(string))
	}

	if v, ok := d.GetOk("partition_count"); ok {
		command.PartitionCount = azure.String(v.(string))
	}

	createRequest := rivieraClient.NewRequest()
	createRequest.Command = command

	createResponse, err := createRequest.Execute()
	if err != nil {
		return fmt.Errorf("Error creating Search Service: %s", err)
	}
	if !createResponse.IsSuccessful() {
		return fmt.Errorf("Error creating Search Service: %s", createResponse.Error)
	}

	readRequest := rivieraClient.NewRequest()
	readRequest.Command = &search.GetSearchService{
		Name:              d.Get("name").(string),
		ResourceGroupName: d.Get("resource_group_name").(string),
	}

	readResponse, err := readRequest.Execute()
	if err != nil {
		return fmt.Errorf("Error reading Search Service: %s", err)
	}
	if !readResponse.IsSuccessful() {
		return fmt.Errorf("Error reading Search Service: %s", readResponse.Error)
	}

	resp := readResponse.Parsed.(*search.GetSearchServiceResponse)
	d.SetId(*resp.ID)

	return resourceArmSearchServiceRead(d, meta)
}

func resourceArmSearchServiceRead(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*ArmClient)
	rivieraClient := client.rivieraClient

	readRequest := rivieraClient.NewRequestForURI(d.Id())
	readRequest.Command = &search.GetSearchService{}

	readResponse, err := readRequest.Execute()
	if err != nil {
		return fmt.Errorf("Error reading Search Service: %s", err)
	}
	if !readResponse.IsSuccessful() {
		log.Printf("[INFO] Error reading Search Service %q - removing from state", d.Id())
		d.SetId("")
		return fmt.Errorf("Error reading Search Service: %s", readResponse.Error)
	}

	resp := readResponse.Parsed.(*search.GetSearchServiceResponse)
	d.Set("sku", resp.Sku)
	return nil
}

func resourceArmSearchServiceDelete(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*ArmClient)
	rivieraClient := client.rivieraClient

	deleteRequest := rivieraClient.NewRequestForURI(d.Id())
	deleteRequest.Command = &search.DeleteSearchService{}

	deleteResponse, err := deleteRequest.Execute()
	if err != nil {
		return fmt.Errorf("Error deleting Search Service: %s", err)
	}
	if !deleteResponse.IsSuccessful() {
		return fmt.Errorf("Error deleting Search Service: %s", deleteResponse.Error)
	}

	return nil
}
