package azure

import (
	"encoding/base64"
	"fmt"
	"log"

	"github.com/Azure/azure-sdk-for-go/management"
	"github.com/Azure/azure-sdk-for-go/management/storageservice"
	"github.com/hashicorp/terraform/helper/schema"
)

// resourceAsmStorageServiceCreate does all the necessary API calls to
// create a new Azure storage service.
func resourceAsmStorageServiceCreate(d *schema.ResourceData, meta interface{}) error {
	azureClient := meta.(*AzureClient)
	mgmtClient := azureClient.asmClient.mgmtClient
	storageServiceClient := azureClient.asmClient.storageServiceClient

	// get all the values:
	log.Println("[INFO] Creating Azure Storage Service creation parameters.")
	name := d.Get("name").(string)
	location := d.Get("location").(string)
	accountType := storageservice.AccountType(d.Get("account_type").(string))
	affinityGroup := d.Get("affinity_group").(string)
	description := d.Get("description").(string)
	label := base64.StdEncoding.EncodeToString([]byte(d.Get("label").(string)))
	var props []storageservice.ExtendedProperty
	if given := d.Get("properties").(map[string]interface{}); len(given) > 0 {
		props = []storageservice.ExtendedProperty{}
		for k, v := range given {
			props = append(props, storageservice.ExtendedProperty{
				Name:  k,
				Value: v.(string),
			})
		}
	}

	// create parameters and send request:
	log.Println("[INFO] Sending Storage Service creation request to Azure.")
	reqID, err := storageServiceClient.CreateStorageService(
		storageservice.StorageAccountCreateParameters{
			ServiceName:   name,
			Location:      location,
			Description:   description,
			Label:         label,
			AffinityGroup: affinityGroup,
			AccountType:   accountType,
			ExtendedProperties: storageservice.ExtendedPropertyList{
				ExtendedProperty: props,
			},
		})
	if err != nil {
		return fmt.Errorf("Failed to create Azure storage service %s: %s", name, err)
	}
	err = mgmtClient.WaitForOperation(reqID, nil)
	if err != nil {
		return fmt.Errorf("Failed creating storage service %s: %s", name, err)
	}

	d.SetId(name)
	return resourceAsmStorageServiceRead(d, meta)
}

// resourceAsmStorageServiceRead does all the necessary API calls to
// read the state of the storage service off Azure.
func resourceAsmStorageServiceRead(d *schema.ResourceData, meta interface{}) error {
	storageServiceClient := meta.(*AzureClient).asmClient.storageServiceClient

	// get our storage service:
	log.Println("[INFO] Sending query about storage service to Azure.")
	name := d.Get("name").(string)
	storsvc, err := storageServiceClient.GetStorageService(name)
	if err != nil {
		if !management.IsResourceNotFoundError(err) {
			return fmt.Errorf("Failed to query about Azure about storage service: %s", err)
		} else {
			// it means that the resource has been deleted from Azure
			// in the meantime and we must remove its associated Resource.
			d.SetId("")
			return nil

		}
	}

	// read values:
	d.Set("url", storsvc.URL)
	log.Println("[INFO] Querying keys of Azure storage service.")
	keys, err := storageServiceClient.GetStorageServiceKeys(name)
	if err != nil {
		return fmt.Errorf("Failed querying keys for Azure storage service: %s", err)
	}
	d.Set("primary_key", keys.PrimaryKey)
	d.Set("secondary_key", keys.SecondaryKey)

	return nil
}

// resourceAsmStorageServiceExists does all the necessary API calls to
// check if the storage service exists on Azure.
func resourceAsmStorageServiceExists(d *schema.ResourceData, meta interface{}) (bool, error) {
	storageServiceClient := meta.(*AzureClient).asmClient.storageServiceClient

	// get our storage service:
	log.Println("[INFO] Sending query about storage service to Azure.")
	name := d.Get("name").(string)
	_, err := storageServiceClient.GetStorageService(name)
	if err != nil {
		if !management.IsResourceNotFoundError(err) {
			return false, fmt.Errorf("Failed to query about Azure about storage service: %s", err)
		} else {
			// it means that the resource has been deleted from Azure
			// in the meantime and we must remove its associated Resource.
			d.SetId("")
			return false, nil

		}
	}

	return true, nil
}

// resourceAsmStorageServiceDelete does all the necessary API calls to
// delete the storage service off Azure.
func resourceAsmStorageServiceDelete(d *schema.ResourceData, meta interface{}) error {
	azureClient := meta.(*AzureClient)
	mgmtClient := azureClient.asmClient.mgmtClient
	storageServiceClient := azureClient.asmClient.storageServiceClient

	// issue the deletion:
	name := d.Get("name").(string)
	log.Println("[INFO] Issuing delete of storage service off Azure.")
	reqID, err := storageServiceClient.DeleteStorageService(name)
	if err != nil {
		return fmt.Errorf("Error whilst issuing deletion of storage service off Azure: %s", err)
	}
	err = mgmtClient.WaitForOperation(reqID, nil)
	if err != nil {
		return fmt.Errorf("Error whilst deleting storage service off Azure: %s", err)
	}

	d.SetId("")
	return nil
}
