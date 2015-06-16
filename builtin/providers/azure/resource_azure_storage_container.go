package azure

import (
	"fmt"
	"log"

	"github.com/Azure/azure-sdk-for-go/storage"
	"github.com/hashicorp/terraform/helper/schema"
)

// resourceAzureStorageContainer returns the *schema.Resource associated
// to a storage container on Azure.
func resourceAzureStorageContainer() *schema.Resource {
	return &schema.Resource{
		Create: resourceAzureStorageContainerCreate,
		Read:   resourceAzureStorageContainerRead,
		Exists: resourceAzureStorageContainerExists,
		Delete: resourceAzureStorageContainerDelete,

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
			"container_access_type": &schema.Schema{
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
				Description: parameterDescriptions["container_access_type"],
			},
			"properties": &schema.Schema{
				Type:        schema.TypeMap,
				Computed:    true,
				Elem:        schema.TypeString,
				Description: parameterDescriptions["properties"],
			},
		},
	}
}

// resourceAzureStorageContainerCreate does all the necessary API calls to
// create the storage container on Azure.
func resourceAzureStorageContainerCreate(d *schema.ResourceData, meta interface{}) error {
	azureClient := meta.(*Client)
	storName := d.Get("storage_service_name").(string)

	blobClient, err := azureClient.getStorageServiceBlobClient(storName)
	if err != nil {
		return err
	}

	log.Println("[INFO] Creating storage container on Azure.")
	name := d.Get("name").(string)
	accessType := storage.ContainerAccessType(d.Get("container_access_type").(string))
	err = blobClient.CreateContainer(name, accessType)
	if err != nil {
		return fmt.Errorf("Failed to create storage container on Azure: %s", err)
	}

	d.SetId(name)
	return resourceAzureStorageContainerRead(d, meta)
}

// resourceAzureStorageContainerRead does all the necessary API calls to
// read the status of the storage container off Azure.
func resourceAzureStorageContainerRead(d *schema.ResourceData, meta interface{}) error {
	azureClient := meta.(*Client)
	storName := d.Get("storage_service_name").(string)

	blobClient, err := azureClient.getStorageServiceBlobClient(storName)
	if err != nil {
		return err
	}

	log.Println("[INFO] Querying Azure for storage containers.")
	name := d.Get("name").(string)
	containers, err := blobClient.ListContainers(storage.ListContainersParameters{
		Prefix:  name,
		Timeout: 90,
	})
	if err != nil {
		return fmt.Errorf("Failed to query Azure for its storage containers: %s", err)
	}

	// search for our storage container and update its stats:
	var found bool
	// loop just to make sure we got the right container:
	for _, cont := range containers.Containers {
		if cont.Name == name {
			found = true

			props := make(map[string]interface{})
			props["last_modified"] = cont.Properties.LastModified
			props["lease_status"] = cont.Properties.LeaseStatus
			props["lease_state"] = cont.Properties.LeaseState
			props["lease_duration"] = cont.Properties.LeaseDuration

			d.Set("properties", props)
		}
	}

	// if not found; it means the resource has been deleted
	// in the meantime; so we must untrack it:
	if !found {
		d.SetId("")
	}

	return nil
}

// resourceAzureStorageContainerExists does all the necessary API calls to
// check if the storage container already exists on Azure.
func resourceAzureStorageContainerExists(d *schema.ResourceData, meta interface{}) (bool, error) {
	azureClient := meta.(*Client)
	storName := d.Get("storage_service_name").(string)

	blobClient, err := azureClient.getStorageServiceBlobClient(storName)
	if err != nil {
		return false, err
	}

	log.Println("[INFO] Checking existence of storage container on Azure.")
	name := d.Get("name").(string)
	exists, err := blobClient.ContainerExists(name)
	if err != nil {
		return false, fmt.Errorf("Failed to query for Azure storage container existence: %s", err)
	}

	// if it does not exist; untrack the resource:
	if !exists {
		d.SetId("")
	}
	return exists, nil
}

// resourceAzureStorageContainerDelete does all the necessary API calls to
// delete a storage container off Azure.
func resourceAzureStorageContainerDelete(d *schema.ResourceData, meta interface{}) error {
	azureClient := meta.(*Client)
	storName := d.Get("storage_service_name").(string)

	blobClient, err := azureClient.getStorageServiceBlobClient(storName)
	if err != nil {
		return err
	}

	log.Println("[INFO] Issuing Azure storage container deletion call.")
	name := d.Get("name").(string)
	if _, err := blobClient.DeleteContainerIfExists(name); err != nil {
		return fmt.Errorf("Failed deleting storage container off Azure: %s", err)
	}

	d.SetId("")
	return nil
}
