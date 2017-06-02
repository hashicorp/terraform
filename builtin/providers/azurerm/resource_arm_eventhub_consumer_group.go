package azurerm

import (
	"fmt"
	"log"

	"net/http"

	"github.com/Azure/azure-sdk-for-go/arm/eventhub"
	"github.com/hashicorp/terraform/helper/schema"
)

func resourceArmEventHubConsumerGroup() *schema.Resource {
	return &schema.Resource{
		Create: resourceArmEventHubConsumerGroupCreateUpdate,
		Read:   resourceArmEventHubConsumerGroupRead,
		Update: resourceArmEventHubConsumerGroupCreateUpdate,
		Delete: resourceArmEventHubConsumerGroupDelete,
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

			"eventhub_name": {
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

			"user_metadata": {
				Type:     schema.TypeString,
				Optional: true,
			},
		},
	}
}

func resourceArmEventHubConsumerGroupCreateUpdate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*ArmClient)
	eventhubClient := client.eventHubConsumerGroupClient
	log.Printf("[INFO] preparing arguments for AzureRM EventHub Consumer Group creation.")

	name := d.Get("name").(string)
	location := d.Get("location").(string)
	namespaceName := d.Get("namespace_name").(string)
	eventHubName := d.Get("eventhub_name").(string)
	resGroup := d.Get("resource_group_name").(string)
	userMetaData := d.Get("user_metadata").(string)

	parameters := eventhub.ConsumerGroupCreateOrUpdateParameters{
		Name:     &name,
		Location: &location,
		ConsumerGroupProperties: &eventhub.ConsumerGroupProperties{
			UserMetadata: &userMetaData,
		},
	}

	_, err := eventhubClient.CreateOrUpdate(resGroup, namespaceName, eventHubName, name, parameters)
	if err != nil {
		return err
	}

	read, err := eventhubClient.Get(resGroup, namespaceName, eventHubName, name)

	if err != nil {
		return err
	}

	if read.ID == nil {
		return fmt.Errorf("Cannot read EventHub Consumer Group %s (resource group %s) ID", name, resGroup)
	}

	d.SetId(*read.ID)

	return resourceArmEventHubConsumerGroupRead(d, meta)
}

func resourceArmEventHubConsumerGroupRead(d *schema.ResourceData, meta interface{}) error {
	eventhubClient := meta.(*ArmClient).eventHubConsumerGroupClient

	id, err := parseAzureResourceID(d.Id())
	if err != nil {
		return err
	}
	resGroup := id.ResourceGroup
	namespaceName := id.Path["namespaces"]
	eventHubName := id.Path["eventhubs"]
	name := id.Path["consumergroups"]

	resp, err := eventhubClient.Get(resGroup, namespaceName, eventHubName, name)
	if err != nil {
		return fmt.Errorf("Error making Read request on Azure EventHub Consumer Group %s: %+v", name, err)
	}
	if resp.StatusCode == http.StatusNotFound {
		d.SetId("")
		return nil
	}

	d.Set("name", name)
	d.Set("location", azureRMNormalizeLocation(*resp.Location))
	d.Set("eventhub_name", eventHubName)
	d.Set("namespace_name", namespaceName)
	d.Set("resource_group_name", resGroup)
	d.Set("user_metadata", resp.ConsumerGroupProperties.UserMetadata)

	return nil
}

func resourceArmEventHubConsumerGroupDelete(d *schema.ResourceData, meta interface{}) error {
	eventhubClient := meta.(*ArmClient).eventHubConsumerGroupClient

	id, err := parseAzureResourceID(d.Id())
	if err != nil {
		return err
	}
	resGroup := id.ResourceGroup
	namespaceName := id.Path["namespaces"]
	eventHubName := id.Path["eventhubs"]
	name := id.Path["consumergroups"]

	resp, err := eventhubClient.Delete(resGroup, namespaceName, eventHubName, name)

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("Error issuing Azure ARM delete request of EventHub Consumer Group '%s': %+v", name, err)
	}

	return nil
}
