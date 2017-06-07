package azurerm

import (
	"fmt"
	"log"

	"net/http"

	"github.com/Azure/azure-sdk-for-go/arm/eventhub"
	"github.com/hashicorp/terraform/helper/schema"
)

func resourceArmEventHub() *schema.Resource {
	return &schema.Resource{
		Create: resourceArmEventHubCreate,
		Read:   resourceArmEventHubRead,
		Update: resourceArmEventHubCreate,
		Delete: resourceArmEventHubDelete,
		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},

		Schema: map[string]*schema.Schema{
			"name": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"namespace_name": {
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

			"partition_count": {
				Type:         schema.TypeInt,
				Required:     true,
				ValidateFunc: validateEventHubPartitionCount,
			},

			"message_retention": {
				Type:         schema.TypeInt,
				Required:     true,
				ValidateFunc: validateEventHubMessageRetentionCount,
			},

			"partition_ids": {
				Type:     schema.TypeSet,
				Elem:     &schema.Schema{Type: schema.TypeString},
				Set:      schema.HashString,
				Computed: true,
			},
		},
	}
}

func resourceArmEventHubCreate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*ArmClient)
	eventhubClient := client.eventHubClient
	log.Printf("[INFO] preparing arguments for Azure ARM EventHub creation.")

	name := d.Get("name").(string)
	location := d.Get("location").(string)
	namespaceName := d.Get("namespace_name").(string)
	resGroup := d.Get("resource_group_name").(string)
	partitionCount := int64(d.Get("partition_count").(int))
	messageRetention := int64(d.Get("message_retention").(int))

	parameters := eventhub.CreateOrUpdateParameters{
		Location: &location,
		Properties: &eventhub.Properties{
			PartitionCount:         &partitionCount,
			MessageRetentionInDays: &messageRetention,
		},
	}

	_, err := eventhubClient.CreateOrUpdate(resGroup, namespaceName, name, parameters)
	if err != nil {
		return err
	}

	read, err := eventhubClient.Get(resGroup, namespaceName, name)
	if err != nil {
		return err
	}

	if read.ID == nil {
		return fmt.Errorf("Cannot read EventHub %s (resource group %s) ID", name, resGroup)
	}

	d.SetId(*read.ID)

	return resourceArmEventHubRead(d, meta)
}

func resourceArmEventHubRead(d *schema.ResourceData, meta interface{}) error {
	eventhubClient := meta.(*ArmClient).eventHubClient

	id, err := parseAzureResourceID(d.Id())
	if err != nil {
		return err
	}
	resGroup := id.ResourceGroup
	namespaceName := id.Path["namespaces"]
	name := id.Path["eventhubs"]

	resp, err := eventhubClient.Get(resGroup, namespaceName, name)
	if err != nil {
		return fmt.Errorf("Error making Read request on Azure EventHub %s: %s", name, err)
	}
	if resp.StatusCode == http.StatusNotFound {
		d.SetId("")
		return nil
	}

	d.Set("name", resp.Name)
	d.Set("location", azureRMNormalizeLocation(*resp.Location))
	d.Set("namespace_name", namespaceName)
	d.Set("resource_group_name", resGroup)

	d.Set("partition_count", resp.Properties.PartitionCount)
	d.Set("message_retention", resp.Properties.MessageRetentionInDays)
	d.Set("partition_ids", resp.Properties.PartitionIds)

	return nil
}

func resourceArmEventHubDelete(d *schema.ResourceData, meta interface{}) error {
	eventhubClient := meta.(*ArmClient).eventHubClient

	id, err := parseAzureResourceID(d.Id())
	if err != nil {
		return err
	}
	resGroup := id.ResourceGroup
	namespaceName := id.Path["namespaces"]
	name := id.Path["eventhubs"]

	resp, err := eventhubClient.Delete(resGroup, namespaceName, name)

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("Error issuing Azure ARM delete request of EventHub'%s': %s", name, err)
	}

	return nil
}

func validateEventHubPartitionCount(v interface{}, k string) (ws []string, errors []error) {
	value := v.(int)

	if !(32 >= value && value >= 2) {
		errors = append(errors, fmt.Errorf("EventHub Partition Count has to be between 2 and 32"))
	}
	return
}

func validateEventHubMessageRetentionCount(v interface{}, k string) (ws []string, errors []error) {
	value := v.(int)

	if !(7 >= value && value >= 1) {
		errors = append(errors, fmt.Errorf("EventHub Retention Count has to be between 1 and 7"))
	}
	return
}
