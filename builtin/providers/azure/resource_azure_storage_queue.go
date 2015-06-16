package azure

import (
	"fmt"
	"log"

	"github.com/hashicorp/terraform/helper/schema"
)

// resourceAzureStorageQueue returns the *schema.Resource associated
// to a storage queue on Azure.
func resourceAzureStorageQueue() *schema.Resource {
	return &schema.Resource{
		Create: resourceAzureStorageQueueCreate,
		Read:   resourceAzureStorageQueueRead,
		Delete: resourceAzureStorageQueueDelete,

		Schema: map[string]*schema.Schema{
			"name": &schema.Schema{
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
				Description: parameterDescriptions["name"],
			},
			"storage_service_name": &schema.Schema{
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
				Description: parameterDescriptions["storage_service_name"],
			},
		},
	}
}

// resourceAzureStorageQueueCreate does all the necessary API calls to
// create a storage queue on Azure.
func resourceAzureStorageQueueCreate(d *schema.ResourceData, meta interface{}) error {
	azureClient := meta.(*Client)
	storServName := d.Get("storage_service_name").(string)
	queueClient, err := azureClient.getStorageServiceQueueClient(storServName)
	if err != nil {
		return err
	}

	// create the queue:
	log.Println("Sending Storage Queue creation request to Azure.")
	name := d.Get("name").(string)
	err = queueClient.CreateQueue(name)
	if err != nil {
		return fmt.Errorf("Error creation Storage Queue on Azure: %s", err)
	}

	d.SetId(name)
	return nil
}

// resourceAzureStorageQueueRead does all the necessary API calls to
// read the state of the storage queue off Azure.
func resourceAzureStorageQueueRead(d *schema.ResourceData, meta interface{}) error {
	azureClient := meta.(*Client)
	storServName := d.Get("storage_service_name").(string)
	queueClient, err := azureClient.getStorageServiceQueueClient(storServName)
	if err != nil {
		return err
	}

	// check for queue's existence:
	log.Println("[INFO] Sending Storage Queue existence query to Azure.")
	name := d.Get("name").(string)
	exists, err := queueClient.QueueExists(name)
	if err != nil {
		return fmt.Errorf("Error checking for Storage Queue existence: %s", err)
	}

	// If the queue has been deleted in the meantime;
	// untrack the resource from the schema.
	if !exists {
		d.SetId("")
	}

	return nil
}

// resourceAzureStorageQueueDelete does all the necessary API calls to
// delete the storage queue off Azure.
func resourceAzureStorageQueueDelete(d *schema.ResourceData, meta interface{}) error {
	azureClient := meta.(*Client)
	storServName := d.Get("storage_service_name").(string)
	queueClient, err := azureClient.getStorageServiceQueueClient(storServName)
	if err != nil {
		return err
	}

	// issue the deletion of the storage queue:
	log.Println("[INFO] Sending Storage Queue deletion request to Azure.")
	name := d.Get("name").(string)
	err = queueClient.DeleteQueue(name)
	if err != nil {
		return fmt.Errorf("Error deleting Storage queue off Azure: %s", err)
	}

	return nil
}
