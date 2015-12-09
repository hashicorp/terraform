package azure

import (
	"fmt"
	"log"

	"github.com/hashicorp/terraform/helper/schema"
)

// resourceAsmStorageQueue returns the *schema.Resource associated
// to a storage queue on Azure.
func resourceAsmStorageQueue() *schema.Resource {
	return &schema.Resource{
		Create: resourceAsmStorageQueueCreate,
		Read:   resourceAsmStorageQueueRead,
		Delete: resourceAsmStorageQueueDelete,

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

// resourceAsmStorageQueueCreate does all the necessary API calls to
// create a storage queue on Azure.
func resourceAsmStorageQueueCreate(d *schema.ResourceData, meta interface{}) error {
	azureClient := meta.(*AzureClient)
	storServName := d.Get("storage_service_name").(string)
	queueClient, err := azureClient.asmClient.getStorageServiceQueueClient(storServName)
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

// resourceAsmStorageQueueRead does all the necessary API calls to
// read the state of the storage queue off Azure.
func resourceAsmStorageQueueRead(d *schema.ResourceData, meta interface{}) error {
	azureClient := meta.(*AzureClient)
	storServName := d.Get("storage_service_name").(string)
	queueClient, err := azureClient.asmClient.getStorageServiceQueueClient(storServName)
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

// resourceAsmStorageQueueDelete does all the necessary API calls to
// delete the storage queue off Azure.
func resourceAsmStorageQueueDelete(d *schema.ResourceData, meta interface{}) error {
	azureClient := meta.(*AzureClient)
	storServName := d.Get("storage_service_name").(string)
	queueClient, err := azureClient.asmClient.getStorageServiceQueueClient(storServName)
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
