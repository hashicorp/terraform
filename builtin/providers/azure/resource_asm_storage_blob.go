package azure

import (
	"fmt"
	"log"

	"github.com/hashicorp/terraform/helper/schema"
)

// resourceAsmStorageBlob returns the *schema.Resource associated
// with a storage blob on Azure.
func resourceAsmStorageBlob() *schema.Resource {
	return &schema.Resource{
		Create: resourceAsmStorageBlobCreate,
		Read:   resourceAsmStorageBlobRead,
		Exists: resourceAsmStorageBlobExists,
		Delete: resourceAsmStorageBlobDelete,

		Schema: map[string]*schema.Schema{
			"name": &schema.Schema{
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
				Description: parameterDescriptions["name"],
			},
			"type": &schema.Schema{
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
				Description: parameterDescriptions["type"],
			},
			"size": &schema.Schema{
				Type:     schema.TypeInt,
				Required: true,
				ForceNew: true,
				DefaultFunc: func() (interface{}, error) {
					return int64(0), nil
				},
			},
			"storage_container_name": &schema.Schema{
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
				Description: parameterDescriptions["storage_container_name"],
			},
			"storage_service_name": {
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
				Description: parameterDescriptions["storage_service_name"],
			},
			"url": &schema.Schema{
				Type:        schema.TypeString,
				Computed:    true,
				Description: parameterDescriptions["url"],
			},
		},
	}
}

// resourceAsmStorageBlobCreate does all the necessary API calls to
// create the storage blob on Azure.
func resourceAsmStorageBlobCreate(d *schema.ResourceData, meta interface{}) error {
	azureClient := meta.(*AzureClient)
	storName := d.Get("storage_service_name").(string)

	blobClient, err := azureClient.asmClient.getStorageServiceBlobClient(storName)
	if err != nil {
		return err
	}

	log.Println("[INFO] Issuing create on Azure storage blob.")
	name := d.Get("name").(string)
	blobType := d.Get("type").(string)
	cont := d.Get("storage_container_name").(string)
	switch blobType {
	case "BlockBlob":
		err = blobClient.CreateBlockBlob(cont, name)
	case "PageBlob":
		size := int64(d.Get("size").(int))
		err = blobClient.PutPageBlob(cont, name, size)
	default:
		err = fmt.Errorf("Invalid blob type specified; see parameter desciptions for more info.")
	}
	if err != nil {
		return fmt.Errorf("Error creating storage blob on Azure: %s", err)
	}

	d.SetId(name)
	return resourceAsmStorageBlobRead(d, meta)
}

// resourceAsmStorageBlobRead does all the necessary API calls to
// read the status of the storage blob off Azure.
func resourceAsmStorageBlobRead(d *schema.ResourceData, meta interface{}) error {
	azureClient := meta.(*AzureClient)

	// check for it's existence:
	exists, err := resourceAsmStorageBlobExists(d, meta)
	if err != nil {
		return err
	}

	// if it exists; read relevant information:
	if exists {
		storName := d.Get("storage_service_name").(string)

		blobClient, err := azureClient.asmClient.getStorageServiceBlobClient(storName)
		if err != nil {
			return err
		}

		name := d.Get("name").(string)
		cont := d.Get("storage_container_name").(string)
		url := blobClient.GetBlobURL(cont, name)
		d.Set("url", url)
	}

	// NOTE: no need to unset the ID here, as resourceAsmStorageBlobExists
	// already should have done so if it were required.
	return nil
}

// resourceAsmStorageBlobExists does all the necessary API calls to
// check for the existence of the blob on Azure.
func resourceAsmStorageBlobExists(d *schema.ResourceData, meta interface{}) (bool, error) {
	azureClient := meta.(*AzureClient)
	storName := d.Get("storage_service_name").(string)

	blobClient, err := azureClient.asmClient.getStorageServiceBlobClient(storName)
	if err != nil {
		return false, err
	}

	log.Println("[INFO] Querying Azure for storage blob's existence.")
	name := d.Get("name").(string)
	cont := d.Get("storage_container_name").(string)
	exists, err := blobClient.BlobExists(cont, name)
	if err != nil {
		return false, fmt.Errorf("Error whilst checking for Azure storage blob's existence: %s", err)
	}

	// if not found; it means it was deleted in the meantime and
	// we must remove it from the schema.
	if !exists {
		d.SetId("")
	}

	return exists, nil
}

// resourceAsmStorageBlobDelete does all the necessary API calls to
// delete the blob off Azure.
func resourceAsmStorageBlobDelete(d *schema.ResourceData, meta interface{}) error {
	azureClient := meta.(*AzureClient)
	storName := d.Get("storage_service_name").(string)

	blobClient, err := azureClient.asmClient.getStorageServiceBlobClient(storName)
	if err != nil {
		return err
	}

	log.Println("[INFO] Issuing storage blob delete command off Azure.")
	name := d.Get("name").(string)
	cont := d.Get("storage_container_name").(string)
	if _, err = blobClient.DeleteBlobIfExists(cont, name); err != nil {
		return fmt.Errorf("Error whilst deleting storage blob: %s", err)
	}

	d.SetId("")
	return nil
}
